package testutils

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"time"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	. "github.com/onsi/gomega"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	operators "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ctiName  = "mycluster"
	ctiNs    = "cluster-aas-operator"
	ctqName  = "myquota"
	ctName   = "mytemplate"
	timeout  = time.Second * 10
	duration = time.Second * 10
	interval = time.Millisecond * 250
)

var ctiSecret = "mysecret"

func GetCTQWithDeletion(deleteAfter time.Duration) *v1alpha1.ClusterTemplateQuota {
	ctq := &v1alpha1.ClusterTemplateQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.APIVersion,
			Kind:       "ClusterTemplateQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctName,
			Namespace: ctiNs,
		},
		Spec: v1alpha1.ClusterTemplateQuotaSpec{
			AllowedTemplates: []v1alpha1.AllowedTemplate{
				{DeleteAfter: &metav1.Duration{Duration: deleteAfter}, Name: ctName, Count: 1},
			},
		},
	}
	return ctq
}

func GetCTQ() *v1alpha1.ClusterTemplateQuota {
	return GetCTQWithDeletion(120 * time.Second)
}

func GetCTIWithSecret() *v1alpha1.ClusterTemplateInstance {
	cti := GetCTI()
	cti.Spec.KubeconfigSecretRef = &ctiSecret
	return cti
}

func GetCTI() *v1alpha1.ClusterTemplateInstance {
	cti := &v1alpha1.ClusterTemplateInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.APIVersion,
			Kind:       "ClusterTemplateInstance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctiName,
			Namespace: ctiNs,
			Finalizers: []string{
				v1alpha1.CTIFinalizer,
			},
		},
		Spec: v1alpha1.ClusterTemplateInstanceSpec{
			ClusterTemplateRef: ctName,
		},
	}
	return cti
}

func GetCTSetup() *v1alpha1.ClusterTemplateSetup {
	ct := &v1alpha1.ClusterTemplateSetup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       "ClusterTemplateSetup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ctName,
		},
		Spec: v1alpha1.ClusterTemplateSetupSpec{
			SkipClusterRegistration: true,
		},
	}
	return ct
}

func GetCTWithCost(withSetup bool, cost *int, skip bool) *v1alpha1.ClusterTemplate {
	ct := &v1alpha1.ClusterTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       "ClusterTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ctName,
		},
		Spec: v1alpha1.ClusterTemplateSpec{
			Cost:                    cost,
			ClusterDefinition:       "appset1",
			SkipClusterRegistration: skip,
		},
	}
	if withSetup {
		ct.Spec.ClusterSetup = []string{"appset2"}
	}
	return ct
}

func GetAppset2() *argo.ApplicationSet {
	return &argo.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       argo.ApplicationSetSchemaGroupVersionKind.Kind,
			APIVersion: argo.ApplicationSetSchemaGroupVersionKind.GroupVersion().Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "appset2",
			Namespace: "cluster-aas-operator",
		},
		Spec: argo.ApplicationSetSpec{
			Generators: []argo.ApplicationSetGenerator{{}},
			Template: argo.ApplicationSetTemplate{
				Spec: argo.ApplicationSpec{
					Source: &argo.ApplicationSource{
						RepoURL:        "http://foo.com",
						TargetRevision: "0.1.0",
						Chart:          "hypershift-template",
					},
				},
			},
		},
	}
}

func GetAppset() *argo.ApplicationSet {
	return &argo.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       argo.ApplicationSetSchemaGroupVersionKind.Kind,
			APIVersion: argo.ApplicationSetSchemaGroupVersionKind.GroupVersion().Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "appset1",
			Namespace: "cluster-aas-operator",
		},
		Spec: argo.ApplicationSetSpec{
			Generators: []argo.ApplicationSetGenerator{{}},
			Template: argo.ApplicationSetTemplate{
				Spec: argo.ApplicationSpec{
					Destination: argo.ApplicationDestination{
						Namespace: "cluster-aas-operator",
					},
					Source: &argo.ApplicationSource{
						RepoURL:        "http://foo.com",
						TargetRevision: "0.1.0",
						Chart:          "hypershift-template",
					},
				},
			},
		},
	}
}

func GetAppDay2() *argo.Application {
	return &argo.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       argo.ApplicationSchemaGroupVersionKind.Kind,
			APIVersion: argo.ApplicationSchemaGroupVersionKind.GroupVersion().Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctiName,
			Namespace: "cluster-aas-operator",
			Labels: map[string]string{
				v1alpha1.CTINameLabel:      ctiName,
				v1alpha1.CTINamespaceLabel: ctiNs,
				v1alpha1.CTISetupLabel:     "",
			},
		},
		Spec: argo.ApplicationSpec{
			Source: &argo.ApplicationSource{
				RepoURL:        "http://foo.com",
				TargetRevision: "0.1.0",
				Chart:          "hypershift-template",
			},
		},
	}
}

func GetApp() *argo.Application {
	return &argo.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       argo.ApplicationSchemaGroupVersionKind.Kind,
			APIVersion: argo.ApplicationSchemaGroupVersionKind.GroupVersion().Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctiName,
			Namespace: "cluster-aas-operator",
			Labels: map[string]string{
				v1alpha1.CTINameLabel:      ctiName,
				v1alpha1.CTINamespaceLabel: ctiNs,
			},
		},
		Spec: argo.ApplicationSpec{
			Source: &argo.ApplicationSource{
				RepoURL:        "http://foo.com",
				TargetRevision: "0.1.0",
				Chart:          "hypershift-template",
			},
		},
	}
}

func GetCT(withSetup bool) *v1alpha1.ClusterTemplate {
	return GetCTWithCost(withSetup, nil, false)
}

func GetSubscription(config *operators.SubscriptionConfig) *operators.Subscription {
	return &operators.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operators.GroupVersion,
			Kind:       operators.SubscriptionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-operator",
			Namespace: "openshift-operators",
			Labels: map[string]string{
				"operators.coreos.com/argocd-operator.openshift-operators": "",
			},
		},
		Spec: &operators.SubscriptionSpec{
			Channel:                "alpha",
			CatalogSource:          "community-operators",
			CatalogSourceNamespace: "openshift-marketplace",
			Package:                "argocd-operator",
			InstallPlanApproval:    operators.ApprovalAutomatic,
			StartingCSV:            "argocd-operator.v0.5.0",
			Config:                 config,
		},
	}
}

func GetKubeconfigSecretWithName(name string) (*corev1.Secret, error) {
	kubeconfigFile, err := ioutil.ReadFile("../testutils/kubeconfig_mock.yaml")
	if err != nil {
		return nil, err
	}
	kubeConfigSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ctiNs,
		},
		Data: map[string][]byte{
			"kubeconfig": kubeconfigFile,
		},
	}
	return kubeConfigSecret, nil
}

func GetKubeconfigSecret() (*corev1.Secret, error) {
	return GetKubeconfigSecretWithName("hypershift-kube-config")
}

func GetKubeadminSecret() (*corev1.Secret, error) {
	kubeAdminBytes, err := json.Marshal("foo")
	if err != nil {
		return nil, err
	}
	kubeAdminSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hypershift-kube-admin-pass",
			Namespace: ctiNs,
		},
		Data: map[string][]byte{
			"password": kubeAdminBytes,
		},
	}
	return kubeAdminSecret, nil
}

func SetHostedClusterReady(
	hostedCluster hypershiftv1beta1.HostedCluster,
	kubeconfigName string,
	kubeadminName string,
) hypershiftv1beta1.HostedCluster {
	status := hypershiftv1beta1.HostedClusterStatus{}

	status.Conditions = []metav1.Condition{
		{
			Type:               string(hypershiftv1beta1.HostedClusterAvailable),
			Status:             metav1.ConditionTrue,
			Reason:             "Foo",
			LastTransitionTime: metav1.Now(),
		},
	}
	status.KubeConfig = &corev1.LocalObjectReference{
		Name: kubeconfigName,
	}
	status.KubeadminPassword = &corev1.LocalObjectReference{
		Name: kubeadminName,
	}
	hostedCluster.Status = status
	hostedCluster.Labels = map[string]string{
		"foo": "bar",
	}
	return hostedCluster
}

func EnsureResourceDoesNotExist(ctx context.Context, obj client.Object, k8sClient client.Client) {
	Eventually(func() bool {
		err := k8sClient.Get(
			ctx,
			types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()},
			obj,
		)
		return apierrors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue())
}

func DeleteResource(ctx context.Context, obj client.Object, k8sClient client.Client) {
	err := k8sClient.Delete(ctx, obj)
	Expect(client.IgnoreNotFound(err)).Should(Succeed())
	EnsureResourceDoesNotExist(ctx, obj, k8sClient)
}
