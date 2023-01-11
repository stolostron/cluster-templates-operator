package bridge

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func CreateTypedClient(token string, config rest.Config) (*kubernetes.Clientset, error) {
	config.BearerToken = token

	return kubernetes.NewForConfig(&config)
}
