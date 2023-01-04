package helm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	argoCommon "github.com/argoproj/argo-cd/v2/common"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getRepoHTTPClient(
	ctx context.Context,
	k8sClient client.Client,
	repoURL string,
	namespace string,
) (*http.Client, error) {
	secrets := &corev1.SecretList{}
	repoLabelReq, _ := labels.NewRequirement(
		argoCommon.LabelKeySecretType,
		selection.Equals,
		[]string{argoCommon.LabelValueSecretTypeRepository},
	)
	selector := labels.NewSelector().Add(*repoLabelReq)
	if err := k8sClient.List(ctx, secrets, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     namespace,
	}); err != nil {
		return nil, err
	}
	tlsClientCertData := []byte{}
	tlsClientCertKey := []byte{}
	insecure := false
	for _, secret := range secrets.Items {
		repoType, repoOk := secret.Data["type"]
		url, urlOk := secret.Data["url"]
		if repoOk && urlOk && (string(repoType) == "helm") && (string(url) == repoURL) {
			certData, certDataOk := secret.Data["tlsClientCertData"]
			if certDataOk {
				tlsClientCertData = certData
			}
			certKey, certKeyOk := secret.Data["tlsClientCertKey"]
			if certKeyOk {
				tlsClientCertKey = certKey
			}
			insecureStr, insecureOk := secret.Data["insecure"]
			if insecureOk && string(insecureStr) == "true" {
				insecure = true
			}
			break
		}
	}

	cm := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: "argocd-tls-certs-cm", Namespace: namespace}, cm); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	}

	parsedUrl, err := url.ParseRequestURI(repoURL)
	if err != nil {
		return nil, err
	}
	var rootCAs *x509.CertPool
	for key, cert := range cm.Data {
		if parsedUrl.Host == key {
			rootCAs = x509.NewCertPool()
			rootCAs.AppendCertsFromPEM([]byte(cert))
			break
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

func (h *HelmClient) GetChart(
	ctx context.Context,
	k8sClient client.Client,
	repoURL string,
	chartName string,
	version string,
	argoCDNamespace string,
) (*chart.Chart, error) {

	httpClient, err := getRepoHTTPClient(ctx, k8sClient, repoURL, argoCDNamespace)

	if err != nil {
		return nil, err
	}

	chartURL, err := getChartURL(
		httpClient,
		repoURL,
		chartName,
		version,
	)

	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Get(chartURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf(
			"response for %v returned %v with status code %v",
			chartURL,
			resp,
			resp.StatusCode,
		)
	}
	defer resp.Body.Close()

	f, err := os.CreateTemp("", "chart-*")
	if err != nil {
		return nil, err
	}

	defer f.Close()
	defer os.Remove(f.Name())

	_, err = io.Copy(f, resp.Body)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return loader.Load(f.Name())
}
