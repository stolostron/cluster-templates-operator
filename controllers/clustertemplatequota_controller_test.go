package controllers

import (
	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/testutils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ClusterTemplateQuota controller", func() {
	Context("Initial ClusterTemplateQuota Status", func() {
		ct := &v1alpha1.ClusterTemplate{}
		ctq := &v1alpha1.ClusterTemplateQuota{}
		cti := &v1alpha1.ClusterTemplateInstance{}
		appset := &argo.ApplicationSet{}
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "argocd",
			},
		}
		cost := 1

		BeforeEach(func() {
			k8sClient.Create(ctx, ns)
			ct = testutils.GetCTWithCost(false, &cost, false)
			Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

			appset = testutils.GetAppset()
			Expect(k8sClient.Create(ctx, appset)).Should(Succeed())

			cti = testutils.GetCTI()

			ctq = testutils.GetCTQ()
			Expect(k8sClient.Create(ctx, ctq)).Should(Succeed())
		})

		AfterEach(func() {
			testutils.DeleteResource(ctx, ctq, k8sClient)
			testutils.DeleteResource(ctx, cti, k8sClient)
			testutils.DeleteResource(ctx, ct, k8sClient)
			testutils.DeleteResource(ctx, appset, k8sClient)
		})

		It("Should count the template if cost specified", func() {
			Expect(k8sClient.Create(ctx, cti)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cti), cti)
				if err != nil {
					return false
				}
				clusterDefinitionCondition := meta.FindStatusCondition(
					cti.Status.Conditions,
					string(v1alpha1.ClusterDefinitionCreated),
				)
				if clusterDefinitionCondition == nil {
					return false
				}
				return clusterDefinitionCondition.Status == metav1.ConditionTrue
			}, timeout, interval).Should(BeTrue())
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ctq), ctq)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ctq.Status.BudgetSpent).Should(Equal(1))
			Expect(len(ctq.Status.TemplateInstances)).Should(Equal(1))
			Expect(ctq.Status.TemplateInstances[0].Name).Should(Equal("mytemplate"))
		})

	})
	Context("Initial ClusterTemplateQuota Status no cost", func() {
		ct := &v1alpha1.ClusterTemplate{}
		ctq := &v1alpha1.ClusterTemplateQuota{}
		cti := &v1alpha1.ClusterTemplateInstance{}
		appset := &argo.ApplicationSet{}
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "argocd",
			},
		}

		BeforeEach(func() {
			k8sClient.Create(ctx, ns)
			ct = testutils.GetCT(false)
			Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

			appset = testutils.GetAppset()
			Expect(k8sClient.Create(ctx, appset)).Should(Succeed())
			cti = testutils.GetCTI()

			ctq = testutils.GetCTQ()
			Expect(k8sClient.Create(ctx, ctq)).Should(Succeed())
		})

		AfterEach(func() {
			testutils.DeleteResource(ctx, ctq, k8sClient)
			testutils.DeleteResource(ctx, cti, k8sClient)
			testutils.DeleteResource(ctx, ct, k8sClient)
			testutils.DeleteResource(ctx, appset, k8sClient)
		})
		It("Should not count the template if cost not specified", func() {
			Expect(k8sClient.Create(ctx, cti)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cti), cti)
				if err != nil {
					return false
				}
				clusterDefinitionCondition := meta.FindStatusCondition(
					cti.Status.Conditions,
					string(v1alpha1.ClusterDefinitionCreated),
				)
				if clusterDefinitionCondition == nil {
					return false
				}
				return clusterDefinitionCondition.Status == metav1.ConditionTrue
			}, timeout, interval).Should(BeTrue())
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ctq), ctq)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ctq.Status.BudgetSpent).Should(Equal(0))
			Expect(len(ctq.Status.TemplateInstances)).Should(Equal(0))
		})
	})
})
