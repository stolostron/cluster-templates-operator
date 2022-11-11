package clusterprovider

import (
	"context"
	"errors"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
)

type ClusterDeploymentProvider struct {
	ClusterDeploymentName      string
	ClusterDeploymentNamespace string
}

func (cd ClusterDeploymentProvider) GetClusterStatus(
	ctx context.Context,
	k8sClient client.Client,
	templateInstance v1alpha1.ClusterTemplateInstance,
) (bool, string, error) {
	clusterDeployment := hivev1.ClusterDeployment{}
	if err := k8sClient.Get(
		ctx,
		client.ObjectKey{Name: cd.ClusterDeploymentName, Namespace: cd.ClusterDeploymentNamespace},
		&clusterDeployment,
	); err != nil {
		return false, "", err
	}

	for _, condition := range clusterDeployment.Status.Conditions {
		if condition.Type == hivev1.ClusterInstallCompletedClusterDeploymentCondition {
			if condition.Status == corev1.ConditionTrue {
				return createCDSecrets(ctx, k8sClient, clusterDeployment, templateInstance)
			} else {
				return false, "Not available - " + condition.Reason, nil
			}
		}
	}
	return false, "Not available", nil
}

type ClusterClaimProvider struct {
	ClusterClaimName      string
	ClusterClaimNamespace string
}

func (cc ClusterClaimProvider) GetClusterStatus(
	ctx context.Context,
	k8sClient client.Client,
	templateInstance v1alpha1.ClusterTemplateInstance,
) (bool, string, error) {
	clusterClaim := hivev1.ClusterClaim{}

	if err := k8sClient.Get(
		ctx,
		client.ObjectKey{Name: cc.ClusterClaimName, Namespace: cc.ClusterClaimNamespace},
		&clusterClaim,
	); err != nil {
		return false, "", err
	}

	if clusterClaim.Spec.Namespace == "" {
		for _, condition := range clusterClaim.Status.Conditions {
			if condition.Type == hivev1.ClusterClaimPendingCondition {
				return false, "Not available - " + condition.Reason, nil
			}
		}
		return false, "Not available", nil
	}

	clusterDeployment := hivev1.ClusterDeployment{}
	if err := k8sClient.Get(
		ctx,
		client.ObjectKey{Name: clusterClaim.Spec.Namespace, Namespace: clusterClaim.Spec.Namespace},
		&clusterDeployment,
	); err != nil {
		return false, "", err
	}

	return createCDSecrets(ctx, k8sClient, clusterDeployment, templateInstance)
}

func getCDKubePassRef(clusterDeployment hivev1.ClusterDeployment) string {
	if clusterDeployment.Spec.ClusterMetadata != nil {
		if clusterDeployment.Spec.ClusterMetadata.AdminPasswordSecretRef != nil {
			return clusterDeployment.Spec.ClusterMetadata.AdminPasswordSecretRef.Name
		}
	}
	return ""
}

func getCDKubeConfigRef(clusterDeployment hivev1.ClusterDeployment) string {
	if clusterDeployment.Spec.ClusterMetadata != nil {
		return clusterDeployment.Spec.ClusterMetadata.AdminKubeconfigSecretRef.Name
	}
	return ""
}

func createCDSecrets(
	ctx context.Context,
	k8sClient client.Client,
	clusterDeployment hivev1.ClusterDeployment,
	templateInstance v1alpha1.ClusterTemplateInstance,
) (bool, string, error) {
	cdKubeAdmin := getCDKubePassRef(clusterDeployment)
	cdKubeConfig := getCDKubeConfigRef(clusterDeployment)
	if cdKubeAdmin == "" || cdKubeConfig == "" {
		return false, "Waiting for pass/kubeconfig secrets", nil
	}

	cdKubeconfigSecret := corev1.Secret{}
	if err := k8sClient.Get(
		ctx,
		client.ObjectKey{Name: cdKubeConfig, Namespace: clusterDeployment.Namespace},
		&cdKubeconfigSecret,
	); err != nil {
		return false, "", err
	}

	kubeconfigBytes, ok := cdKubeconfigSecret.Data["kubeconfig"]

	if !ok {
		return false, "", errors.New("unexpected kubeconfig format")
	}

	cdKubeadminSecret := corev1.Secret{}
	if err := k8sClient.Get(
		ctx,
		client.ObjectKey{Name: cdKubeAdmin, Namespace: clusterDeployment.Namespace},
		&cdKubeadminSecret,
	); err != nil {
		return false, "", err
	}

	username, ok := cdKubeadminSecret.Data["username"]
	if !ok {
		return false, "", errors.New("unexpected kubeadmin format")
	}

	password, ok := cdKubeadminSecret.Data["password"]
	if !ok {
		return false, "", errors.New("unexpected kubeadmin format")
	}

	if err := CreateClusterSecrets(
		ctx,
		k8sClient,
		kubeconfigBytes,
		username,
		password,
		templateInstance,
	); err != nil {
		return false, "", err
	}
	return true, "Available", nil
}
