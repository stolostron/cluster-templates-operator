package hypershift

import (
	"context"

	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeSecret struct {
	Namespace    string
	PassSecret   string
	ConfigSecret string
}

func GetHypershiftInfo(ctx context.Context, manifest string, k8sClient client.Client) (*KubeSecret, string, error) {
	var hostedC hypershiftv1alpha1.HostedCluster
	err := k8sYaml.Unmarshal([]byte(manifest), &hostedC)
	if err != nil {
		return nil, "", err
	}

	hostedCluster := &hypershiftv1alpha1.HostedCluster{}
	err = k8sClient.Get(
		ctx,
		client.ObjectKey{Name: hostedC.Name, Namespace: hostedC.Namespace},
		hostedCluster,
	)

	status := "Not available"
	if err == nil {
		for _, condition := range hostedCluster.Status.Conditions {
			if condition.Type == "Available" {
				if condition.Status == "True" {
					status = "Available"
				} else {
					status = "Not available - " + condition.Reason
				}
			}
		}
	} else {
		return nil, "", err
	}

	return &KubeSecret{
		Namespace:    hostedC.Namespace,
		PassSecret:   hostedC.Name + "-kubeadmin-password", //TODO load from status
		ConfigSecret: hostedC.Name + "-admin-kubeconfig",   //TODO load from status
	}, status, nil
}
