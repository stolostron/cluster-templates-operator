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
	if err := k8sClient.Get(
		ctx,
		client.ObjectKey{Name: hc.HostedClusterName, Namespace: hc.HostedClusterNamespace},
		hostedCluster,
	); err != nil {
		return false, "", err
	}

	availableCondition := metav1.Condition{}

	for _, condition := range hostedCluster.Status.Conditions {
		if condition.Type == string(hypershiftv1alpha1.HostedClusterAvailable) {
			availableCondition = condition
		}
	}

	if availableCondition.Status != metav1.ConditionTrue {
		msg := availableCondition.Message
		if msg == "" {
			msg = availableCondition.Reason
		}
		if msg == "" {
			return false, "Not available", nil
		}
		return false, "Not available - " + msg, nil
	}

	hypershiftPass := getKubeAdminRef(*hostedCluster)
	hypershiftKubeConfig := getKubeConfigRef(*hostedCluster)

	if hypershiftKubeConfig == "" {
		return false, "Waiting for pass/kubeconfig secrets", nil
	}

	hypershiftKubeconfigSecret := corev1.Secret{}
	if err := k8sClient.Get(
		ctx,
		client.ObjectKey{
			Name:      hypershiftKubeConfig,
			Namespace: hostedCluster.Namespace,
		},
		&hypershiftKubeconfigSecret,
	); err != nil {
		return false, "", err
	}

	kubeconfigBytes, ok := hypershiftKubeconfigSecret.Data["kubeconfig"]

	if !ok {
		return false, "", errors.New("unexpected kubeconfig format")
	}

	kubeadminPass := []byte("")

	// if custom idp is configured, kubeadmin pass is not defined
	if !hasIDPs(hostedCluster) {
		if hypershiftPass == "" {
			return false, "Waiting for pass/kubeconfig secrets", nil
		}
		hypershiftKubeadminSecret := corev1.Secret{}
		if err := k8sClient.Get(
			ctx,
			client.ObjectKey{Name: hypershiftPass, Namespace: hostedCluster.Namespace},
			&hypershiftKubeadminSecret,
		); err != nil {
			return false, "", err
		}

		var ok bool
		kubeadminPass, ok = hypershiftKubeadminSecret.Data["password"]

		if !ok {
			return false, "", errors.New("unexpected kubeadmin password format")
		}
	}

	if err := CreateClusterSecrets(
		ctx,
		k8sClient,
		kubeconfigBytes,
		[]byte("kubeadmin"),
		kubeadminPass,
		templateInstance,
	); err != nil {
		return false, "", err
	}

	if len(hc.NodePoolNames) > 0 {
		nodePools := &hypershiftv1alpha1.NodePoolList{}
		if err := k8sClient.List(ctx, nodePools, &client.ListOptions{Namespace: hc.HostedClusterNamespace}); err != nil {
			return false, "", err
		}

		allReady := true
		for _, nodePool := range nodePools.Items {
			if nodePool.Spec.ClusterName == hc.HostedClusterName {
				conditionFound := false
				for _, condition := range nodePool.Status.Conditions {
					if condition.Type == string(
						hypershiftv1alpha1.NodePoolReadyConditionType,
					) {
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

func hasIDPs(hostedCluster *hypershiftv1alpha1.HostedCluster) bool {
	if hostedCluster.Spec.Configuration != nil && hostedCluster.Spec.Configuration.OAuth != nil && hostedCluster.Spec.Configuration.OAuth.IdentityProviders != nil {
		return len(hostedCluster.Spec.Configuration.OAuth.IdentityProviders) > 0
	}
	return false
}
