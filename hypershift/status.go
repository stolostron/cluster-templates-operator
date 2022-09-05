package hypershift

import (
	"context"

	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeSecret struct {
	Kubeadmin  string
	Kubeconfig string
}

func GetHypershiftInfo(
	ctx context.Context,
	manifest string,
	k8sClient client.Client,
) (KubeSecret, bool, string, error) {
	var hostedC hypershiftv1alpha1.HostedCluster
	kubeSecret := KubeSecret{}
	err := k8sYaml.Unmarshal([]byte(manifest), &hostedC)
	if err != nil {
		return kubeSecret, false, "", err
	}

	hostedCluster := &hypershiftv1alpha1.HostedCluster{}
	err = k8sClient.Get(
		ctx,
		client.ObjectKey{Name: hostedC.Name, Namespace: hostedC.Namespace},
		hostedCluster,
	)

	if err != nil {
		return kubeSecret, false, "", err
	}

	status := "Not available"
	ready := false
	for _, condition := range hostedCluster.Status.Conditions {
		if condition.Type == "Available" {
			if condition.Status == "True" {
				status = "Available"
				ready = true
			} else {
				status = "Not available - " + condition.Reason
			}
		}
	}

	if hostedCluster.Status.KubeadminPassword != nil && hostedCluster.Status.KubeConfig != nil {
		kubeSecret.Kubeadmin = hostedCluster.Status.KubeadminPassword.Name
		kubeSecret.Kubeconfig = hostedCluster.Status.KubeConfig.Name
	}

	return kubeSecret, ready, status, nil
}
