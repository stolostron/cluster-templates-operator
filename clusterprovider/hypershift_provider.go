package clusterprovider

import (
	"context"
	"errors"

	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HostedClusterProvider struct {
	HostedClusterName      string
	HostedClusterNamespace string
	NodePoolNames          []string
}

func (hc HostedClusterProvider) GetClusterStatus(
	ctx context.Context,
	k8sClient client.Client,
	templateInstance v1alpha1.ClusterTemplateInstance,
) (bool, string, error) {
	hostedCluster := &hypershiftv1alpha1.HostedCluster{}
	err := k8sClient.Get(
		ctx,
		client.ObjectKey{Name: hc.HostedClusterName, Namespace: hc.HostedClusterNamespace},
		hostedCluster,
	)

	if err != nil {
		return false, "", err
	}

	for _, condition := range hostedCluster.Status.Conditions {
		if condition.Type == string(hypershiftv1alpha1.HostedClusterAvailable) {
			if condition.Status == metav1.ConditionTrue {
				hypershiftPass := getKubeAdminRef(*hostedCluster)
				hypershiftKubeConfig := getKubeConfigRef(*hostedCluster)

				if hypershiftPass == "" || hypershiftKubeConfig == "" {
					return false, "Waiting for pass/kubeconfig secrets", nil
				}

				hypershiftKubeconfigSecret := corev1.Secret{}
				err := k8sClient.Get(
					ctx,
					client.ObjectKey{
						Name:      hypershiftKubeConfig,
						Namespace: hostedCluster.Namespace,
					},
					&hypershiftKubeconfigSecret,
				)

				if err != nil {
					return false, "", err
				}

				kubeconfigBytes, ok := hypershiftKubeconfigSecret.Data["kubeconfig"]

				if !ok {
					return false, "", errors.New("unexpected kubeconfig format")
				}

				hypershiftKubeadminSecret := corev1.Secret{}
				err = k8sClient.Get(
					ctx,
					client.ObjectKey{Name: hypershiftPass, Namespace: hostedCluster.Namespace},
					&hypershiftKubeadminSecret,
				)

				if err != nil {
					return false, "", err
				}

				kubeadminPass, ok := hypershiftKubeadminSecret.Data["password"]

				if !ok {
					return false, "", errors.New("unexpected kubeadmin password format")
				}

				err = CreateClusterSecrets(
					ctx,
					k8sClient,
					kubeconfigBytes,
					[]byte("kubeadmin"),
					kubeadminPass,
					templateInstance,
				)
				if err != nil {
					return false, "", err
				}

				if len(hc.NodePoolNames) > 0 {
					nodePools := &hypershiftv1alpha1.NodePoolList{}
					err := k8sClient.List(ctx, nodePools, &client.ListOptions{Namespace: hc.HostedClusterNamespace})
					if err != nil {
						return false, "", err
					}

					allReady := true
					for _, nodePool := range nodePools.Items {
						if nodePool.Spec.ClusterName == hc.HostedClusterName {
							conditionFound := false
							for _, condition := range nodePool.Status.Conditions {
								if condition.Type == string(hypershiftv1alpha1.NodePoolReadyConditionType) {
									conditionFound = true
									if condition.Status == corev1.ConditionFalse {
										allReady = false
									}
								}
							}
							if !conditionFound {
								allReady = false
							}
						}
					}
					if !allReady {
						return false, "Not available, waiting for nodepools", nil
					}
				}
				return true, "Available", nil
			} else {
				return false, "Not available - " + condition.Reason, nil
			}
		}
	}

	return false, "Not available", nil
}

func getKubeAdminRef(hostedCluster hypershiftv1alpha1.HostedCluster) string {
	if hostedCluster.Status.KubeadminPassword != nil {
		return hostedCluster.Status.KubeadminPassword.Name
	}
	return ""
}

func getKubeConfigRef(hostedCluster hypershiftv1alpha1.HostedCluster) string {
	if hostedCluster.Status.KubeConfig != nil {
		return hostedCluster.Status.KubeConfig.Name
	}
	return ""
}
