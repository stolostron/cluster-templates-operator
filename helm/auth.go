package helm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openshiftAPI "github.com/openshift/api/helm/v1beta1"
)

const (
	configNamespace  = "openshift-config"
	tlsSecretCertKey = "tls.crt"
	tlsSecretKey     = "tls.key"
	tlsSecretPattern = "tlscrt-*"
	tlsKeyPattern    = "tlskey-*"
	caBundleKey      = "ca-bundle.crt"
	cacertPattern    = "cacert-*"
)

func getTlsCert(ctx context.Context, secretName string, k8sClient client.Client) ([]byte, []byte, error) {
	//set up tls cert and key
	secret := corev1.Secret{}
	err := k8sClient.Get(ctx,
		client.ObjectKey{
			Name:      secretName,
			Namespace: configNamespace,
		},
		&secret)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to GET secret %q from %v reason %v", secretName, configNamespace, err)
	}
	tlsCertBytes, found := secret.Data[tlsSecretCertKey]
	if !found {
		return nil, nil, fmt.Errorf("Failed to find %q key in secret %q", tlsSecretCertKey, secretName)
	}
	/*
		tlsCertFile, err := writeTempFile((tlsCertBytes), tlsSecretPattern)
		if err != nil {
			return nil, nil, err
		}
	*/
	tlsKeyBytes, found := secret.Data[tlsSecretKey]
	if !found {
		return nil, nil, fmt.Errorf("Failed to find %q key in secret %q", tlsSecretKey, secretName)
	}
	/*
		tlsKeyFile, err := writeTempFile(tlsKeyBytes, tlsKeyPattern)
		if err != nil {
			return nil, nil, err
		}
	*/

	if err != nil {
		return nil, nil, err
	}

	return tlsCertBytes, tlsKeyBytes, err
}

func getCaCert(ctx context.Context, cacert string, k8sClient client.Client) ([]byte, error) {
	configMap := corev1.ConfigMap{}
	err := k8sClient.Get(ctx,
		client.ObjectKey{
			Name:      cacert,
			Namespace: configNamespace,
		},
		&configMap)
	if err != nil {
		return nil, fmt.Errorf("Failed to GET configmap %q, reason %v", cacert, err)
	}
	caCertBytes, found := configMap.Data[caBundleKey]
	if !found {
		return nil, fmt.Errorf("Failed to find %q key in configmap %q", caBundleKey, cacert)
	}
	return []byte(caCertBytes), nil
}

func GetTLSClientConfig(
	ctx context.Context,
	k8sClient client.Client,
	connectionCofig openshiftAPI.ConnectionConfig,
) (*tls.Config, error) {
	if connectionCofig.TLSClientConfig.Name == "" {
		return nil, nil
	}

	tlsCertBytes, tlsKeyBytes, err := getTlsCert(ctx, connectionCofig.TLSClientConfig.Name, k8sClient)

	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(tlsCertBytes, tlsKeyBytes)

	tlsConfig := tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	if connectionCofig.CA.Name != "" {
		caCert, err := getCaCert(ctx, connectionCofig.CA.Name, k8sClient)

		if err != nil {
			return nil, err
		}

		rootCAs := x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(caCert)

		tlsConfig.RootCAs = rootCAs
	}

	return &tlsConfig, nil
}

// writeTempFile creates a temporary file with the given `data`. `pattern`
// is used by `os.CreateTemp` to create a file in the filesystem.
func writeTempFile(data []byte, pattern string) (*os.File, error) {
	f, createErr := os.CreateTemp("", pattern)
	if createErr != nil {
		return nil, createErr
	}

	_, writeErr := f.Write(data)
	if writeErr != nil {
		return nil, writeErr
	}

	closeErr := f.Close()
	if closeErr != nil {
		return nil, closeErr
	}

	return f, nil
}
