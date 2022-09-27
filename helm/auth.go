package helm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openshiftAPI "github.com/openshift/api/helm/v1beta1"
)

const (
	configNamespace  = "openshift-config"
	tlsSecretCertKey = "tls.crt"
	tlsSecretKey     = "tls.key"
	caBundleKey      = "ca-bundle.crt"
)

func (h *HelmClient) getTlsCert(
	ctx context.Context,
	secretName string,
) ([]byte, []byte, error) {
	//set up tls cert and key
	secret := corev1.Secret{}
	err := h.k8sClient.Get(ctx,
		client.ObjectKey{
			Name:      secretName,
			Namespace: configNamespace,
		},
		&secret)

	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to GET secret %q from %v reason %v",
			secretName,
			configNamespace,
			err,
		)
	}
	tlsCertBytes, found := secret.Data[tlsSecretCertKey]
	if !found {
		return nil, nil, fmt.Errorf(
			"failed to find %q key in secret %q",
			tlsSecretCertKey,
			secretName,
		)
	}

	tlsKeyBytes, found := secret.Data[tlsSecretKey]
	if !found {
		return nil, nil, fmt.Errorf("failed to find %q key in secret %q", tlsSecretKey, secretName)
	}

	if err != nil {
		return nil, nil, err
	}

	return tlsCertBytes, tlsKeyBytes, err
}

func (h *HelmClient) getCaCert(ctx context.Context, cacert string) ([]byte, error) {
	configMap := corev1.ConfigMap{}
	err := h.k8sClient.Get(ctx,
		client.ObjectKey{
			Name:      cacert,
			Namespace: configNamespace,
		},
		&configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to GET configmap %q, reason %v", cacert, err)
	}
	caCertBytes, found := configMap.Data[caBundleKey]
	if !found {
		return nil, fmt.Errorf("failed to find %q key in configmap %q", caBundleKey, cacert)
	}
	return []byte(caCertBytes), nil
}

func (h *HelmClient) GetTLSClientConfig(
	ctx context.Context,
	connectionCofig openshiftAPI.ConnectionConfig,
) (*tls.Config, error) {
	if connectionCofig.TLSClientConfig.Name == "" {
		return nil, nil
	}

	tlsCertBytes, tlsKeyBytes, err := h.getTlsCert(
		ctx,
		connectionCofig.TLSClientConfig.Name,
	)

	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(tlsCertBytes, tlsKeyBytes)

	if err != nil {
		return nil, err
	}

	tlsConfig := tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	if connectionCofig.CA.Name != "" {
		caCert, err := h.getCaCert(ctx, connectionCofig.CA.Name)

		if err != nil {
			return nil, err
		}

		rootCAs := x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(caCert)

		tlsConfig.RootCAs = rootCAs
	}

	return &tlsConfig, nil
}
