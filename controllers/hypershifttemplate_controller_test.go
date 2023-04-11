package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("HypershiftTemplate controller", func() {
	It("Creates default templates", func() {
		for template := range defaultTemplates {
			ct := &v1alpha1.ClusterTemplate{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: template}, ct)
			}, timeout, interval).Should(BeNil())
			//Expect(err).Should(BeNil())
			Expect(ct.Name).Should(Equal(template))
		}
	})
	It("Recreates default templates", func() {
		for template := range defaultTemplates {
			ct := &v1alpha1.ClusterTemplate{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: template}, ct)
			Expect(err).Should(BeNil())
			cost := 500
			ct.Spec.Cost = &cost
			err = k8sClient.Update(ctx, ct)
			Expect(err).Should(BeNil())
			Eventually(func() bool {
				err := k8sClient.Get(
					ctx,
					client.ObjectKeyFromObject(ct),
					ct,
				)
				Expect(err).Should(BeNil())
				return *ct.Spec.Cost == 1
			}, timeout, interval).Should(BeTrue())
		}
	})
})
