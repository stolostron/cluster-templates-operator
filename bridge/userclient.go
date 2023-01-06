package bridge

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateTypedClient(token string, config rest.Config) (*kubernetes.Clientset, error) {
	config.BearerToken = token

	return kubernetes.NewForConfig(&config)
}

func GetClientForClient(token string, config rest.Config) (client.Client, error) {
	config.BearerToken = token

	return client.New(&config, client.Options{})
}
