package repository

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

const (
	RepoSecretTLSClientKey  = "tlsClientCertKey"
	RepoSecretTLSClientCert = "tlsClientCertData"
	RepoSecretUsername      = "username"
	RepoSecretPassword      = "password"
	RepoSecretTLSInsecure   = "insecure"
)

func initSettings() *cli.EnvSettings {
	conf := cli.New()
	conf.RepositoryCache = "/tmp"
	return conf
}

type HttpClient struct {
	secret *corev1.Secret
	client *http.Client
}

func (c *HttpClient) Get(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var username []byte
	var password []byte
	if c.secret != nil {
		if usernameSecret, usernameOk := c.secret.Data[RepoSecretUsername]; usernameOk {
			username = usernameSecret
		}
		if passwordSecret, passwordOk := c.secret.Data[RepoSecretPassword]; passwordOk {
			password = passwordSecret
		}
	}

	if username != nil && password != nil {
		req.SetBasicAuth(string(username), string(password))
	}

	return c.client.Do(req)

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
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: RepoCMName, Namespace: argoCDNamespace}, cm); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	}
	return cm, nil
}

func GetRepoSecret(
	ctx context.Context,
	k8sClient client.Client,
	argoCDNamespace string,
	repoURL string,
) (*corev1.Secret, error) {
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

	var repoSecret *corev1.Secret
	for _, secret := range helmRepoSecrets {
		url, urlOk := secret.Data["url"]
		if urlOk && string(url) == repoURL {
			repoSecret = &secret
			break
		}
	}
	return repoSecret, nil
}

func GetRepoHTTPClient(
	repoURL string,
	repoSecret *corev1.Secret,
	tlsCM *corev1.ConfigMap,
) (*HttpClient, error) {

	tlsClientCertData := []byte{}
	tlsClientCertKey := []byte{}
	insecure := false
	if repoSecret != nil {
		certData, certDataOk := repoSecret.Data[RepoSecretTLSClientCert]
		if certDataOk {
			tlsClientCertData = certData
		}
		certKey, certKeyOk := repoSecret.Data[RepoSecretTLSClientKey]
		if certKeyOk {
			tlsClientCertKey = certKey
		}
		insecureStr, insecureOk := repoSecret.Data[RepoSecretTLSInsecure]
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
			if parsedUrl.Hostname() == key {
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

	client := &HttpClient{
		secret: repoSecret,
		client: httpClient,
	}

	return client, nil
}
