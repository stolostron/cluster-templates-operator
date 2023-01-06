package helm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"

	argoCommon "github.com/argoproj/argo-cd/v2/common"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func initSettings() *cli.EnvSettings {
	conf := cli.New()
	conf.RepositoryCache = "/tmp"
	return conf
}

type HelmClient struct {
	*genericclioptions.ConfigFlags
	config       *rest.Config
	actionConfig *action.Configuration
	k8sClient    client.Client
}

func NewHelmClient(
	config *rest.Config,
	k8sClient client.Client,
	certDataFileName *string,
	keyDataFileName *string,
	caDataFileName *string,
) *HelmClient {
	initSettings()
	ns := "default"

	configFlags := &genericclioptions.ConfigFlags{
		APIServer:   &config.Host,
		BearerToken: &config.BearerToken,
		Namespace:   &ns,
		CAFile:      &config.CAFile,
	}

	if caDataFileName != nil {
		configFlags.CAFile = caDataFileName
	}
	if certDataFileName != nil {
		configFlags.CertFile = certDataFileName
	}
	if keyDataFileName != nil {
		configFlags.KeyFile = keyDataFileName
	}

	helmClient := HelmClient{
		ConfigFlags: configFlags,
		config:      config,
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(helmClient, ns, "secrets", klog.Infof); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	helmClient.actionConfig = actionConfig
	helmClient.k8sClient = k8sClient

	return &helmClient
}

const RepoCMName = "argocd-tls-certs-cm"

func GetRepoCM(
	ctx context.Context,
	k8sClient client.Client,
	argoCDNamespace string,
) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: "argocd-tls-certs-cm", Namespace: argoCDNamespace}, cm); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	}
	return cm, nil
}

func GetRepoSecrets(
	ctx context.Context,
	k8sClient client.Client,
	argoCDNamespace string,
) ([]corev1.Secret, error) {
	secrets := &corev1.SecretList{}
	repoLabelReq, _ := labels.NewRequirement(
		argoCommon.LabelKeySecretType,
		selection.Equals,
		[]string{argoCommon.LabelValueSecretTypeRepository},
	)
	selector := labels.NewSelector().Add(*repoLabelReq)
	if err := k8sClient.List(ctx, secrets, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     argoCDNamespace,
	}); err != nil {
		return nil, err
	}

	helmRepoSecrets := []corev1.Secret{}
	for _, secret := range secrets.Items {
		repoType, repoOk := secret.Data["type"]
		url, urlOk := secret.Data["url"]
		if repoOk && string(repoType) == "helm" && urlOk && string(url) != "" {
			helmRepoSecrets = append(helmRepoSecrets, secret)
		}
	}
	return helmRepoSecrets, nil
}

func GetRepoHTTPClient(
	ctx context.Context,
	repoURL string,
	repoSecrets []corev1.Secret,
	tlsCM *corev1.ConfigMap,
) (*http.Client, error) {

	var repoSecret *corev1.Secret
	for _, secret := range repoSecrets {
		url, urlOk := secret.Data["url"]
		if urlOk && string(url) == repoURL {
			repoSecret = &secret
			break
		}
	}

	tlsClientCertData := []byte{}
	tlsClientCertKey := []byte{}
	insecure := false
	if repoSecret != nil {
		certData, certDataOk := repoSecret.Data["tlsClientCertData"]
		if certDataOk {
			tlsClientCertData = certData
		}
		certKey, certKeyOk := repoSecret.Data["tlsClientCertKey"]
		if certKeyOk {
			tlsClientCertKey = certKey
		}
		insecureStr, insecureOk := repoSecret.Data["insecure"]
		if insecureOk && string(insecureStr) == "true" {
			insecure = true
		}
	}

	parsedUrl, err := url.ParseRequestURI(repoURL)
	if err != nil {
		return nil, err
	}
	var rootCAs *x509.CertPool
	if tlsCM != nil {
		for key, cert := range tlsCM.Data {
			if parsedUrl.Host == key {
				rootCAs = x509.NewCertPool()
				rootCAs.AppendCertsFromPEM([]byte(cert))
				break
			}
		}
	}

	tlsConfig := &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: insecure,
	}

	if len(tlsClientCertData) > 0 && len(tlsClientCertKey) > 0 {
		cert, err := tls.X509KeyPair(tlsClientCertData, tlsClientCertKey)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}

		/*
			getcc := func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return &tlsConfig.Certificates[0], nil
			}

			tlsConfig.GetClientCertificate = getcc
		*/
	}

	httpClient := &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}}

	return httpClient, nil
}
