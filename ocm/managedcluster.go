package ocm

import (
	"context"
	"strings"

	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	ocm "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	utils "github.com/stolostron/cluster-templates-operator/utils"
)

type MCNotFoundError struct{}

func (m *MCNotFoundError) Error() string {
	return "Managed cluster not found"
}

func CreateManagedCluster(
	ctx context.Context,
	k8sClient client.Client,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	labels := map[string]string{
		v1alpha1.CTINameLabel:      clusterTemplateInstance.Name,
		v1alpha1.CTINamespaceLabel: clusterTemplateInstance.Namespace,
		"cloud":                    "auto-detect",
		"vendor":                   "auto-detect",
	}
	for k, v := range clusterTemplateInstance.Status.ClusterTemplateLabels {
		labels[k] = v
	}
	mc := &ocm.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "cluster-",
			Labels:       labels,
		},
		Spec: ocm.ManagedClusterSpec{
			HubAcceptsClient: true,
		},
	}
	return k8sClient.Create(ctx, mc)
}

func GetManagedCluster(
	ctx context.Context,
	k8sClient client.Client,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) (*ocm.ManagedCluster, error) {
	mcs := &ocm.ManagedClusterList{}

	ctiNameLabelReq, _ := labels.NewRequirement(
		v1alpha1.CTINameLabel,
		selection.Equals,
		[]string{clusterTemplateInstance.Name},
	)
	ctiNsLabelReq, _ := labels.NewRequirement(
		v1alpha1.CTINamespaceLabel,
		selection.Equals,
		[]string{clusterTemplateInstance.Namespace},
	)
	selector := labels.NewSelector().Add(*ctiNameLabelReq, *ctiNsLabelReq)

	if err := k8sClient.List(ctx, mcs, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return nil, err
	}

	for _, mc := range mcs.Items {
		if strings.HasPrefix(mc.Name, "cluster-") {
			return &mc, nil
		}
	}
	return nil, nil
}

func ImportManagedCluster(
	ctx context.Context,
	k8sClient client.Client,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) (bool, error) {
	createdMC, err := GetManagedCluster(ctx, k8sClient, clusterTemplateInstance)
	if err != nil {
		return false, err
	}
	if createdMC == nil {
		return false, &MCNotFoundError{}
	}

	kubeconfigSecret := corev1.Secret{}
	if err := k8sClient.Get(
		ctx,
		client.ObjectKey{
			Name:      clusterTemplateInstance.GetKubeconfigRef(),
			Namespace: clusterTemplateInstance.Namespace,
		},
		&kubeconfigSecret,
	); err != nil {
		return false, err
	}

	secret := &corev1.Secret{
		ObjectMeta: GetImportSecretMeta(createdMC.Name),
		Data: map[string][]byte{
			"autoImportRetry": []byte("2"),
			"kubeconfig":      []byte(kubeconfigSecret.Data["kubeconfig"]),
		},
		Type: corev1.SecretTypeOpaque,
	}
	if err := utils.EnsureResourceExists(ctx, k8sClient, secret, false); err != nil {
		return false, err
	}

	imported := false
	for _, c := range createdMC.Status.Conditions {
		if c.Type == "ManagedClusterImportSucceeded" {
			imported = c.Status == metav1.ConditionTrue
		}
	}

	return imported, nil
}

func GetImportSecretMeta(mcName string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      "auto-import-secret",
		Namespace: mcName,
	}
}
