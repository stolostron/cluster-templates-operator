package controllers

import (
	argooperator "github.com/argoproj-labs/argocd-operator/api/v1alpha1"
	operators "github.com/operator-framework/api/pkg/operators/v1alpha1"
	testutils "github.com/stolostron/cluster-templates-operator/testutils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArgoCD controller", func() {
	argo := &argooperator.ArgoCD{}

	AfterEach(func() {
		testutils.DeleteResource(ctx, argo, k8sClient)
	})

	It("Default ArgoCD should be eventually created", func() {
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: argoname, Namespace: defaultArgoCDNs}, argo)
		}, timeout, interval).Should(BeNil())

		Eventually(func() string {
			sub := &operators.Subscription{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: "argocd-operator", Namespace: "openshift-operators"}, sub); err != nil {
				return ""
			}
			if sub.Spec.Config == nil {
				return ""
			}
			return sub.Spec.Config.Env[0].Value
		}, timeout, interval).Should(Equal(defaultArgoCDNs))
	})

})

var _ = Describe("ArgoCD controller with cm", func() {
	argo := &argooperator.ArgoCD{}
	var cm *v1.ConfigMap

	AfterEach(func() {
		testutils.DeleteResource(ctx, cm, k8sClient)
	})

	It("ArgoCD should be deleted if default namespace is changed", func() {
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: argoname, Namespace: defaultArgoCDNs}, argo)
		}, timeout, interval).Should(BeNil())

		Eventually(func() string {
			sub := &operators.Subscription{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: "argocd-operator", Namespace: "openshift-operators"}, sub); err != nil {
				return ""
			}
			if sub.Spec.Config == nil {
				return ""
			}
			return sub.Spec.Config.Env[0].Value
		}, timeout, interval).Should(Equal(defaultArgoCDNs))

		cm = &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claas-config",
				Namespace: "cluster-aas-operator",
			},
			Data: map[string]string{
				argoCDNsConfig: "my-argocd-ns",
			},
		}
		Expect(k8sClient.Create(ctx, cm)).ToNot(HaveOccurred())

		Eventually(func() bool {
			return ArgoCDNamespace == "my-argocd-ns"
		}, timeout, interval).Should(BeTrue())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: argoname, Namespace: defaultArgoCDNs}, argo)
		}, timeout, interval).ShouldNot(BeNil())
		Eventually(func() string {
			sub := &operators.Subscription{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: "argocd-operator", Namespace: "openshift-operators"}, sub); err != nil {
				return "ab"
			}
			if sub.Spec.Config == nil {
				return "a"
			}
			return sub.Spec.Config.Env[0].Value
		}, timeout, interval).Should(Equal(""))
	})

})
