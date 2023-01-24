package defaultresources

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("HypershiftTemplate controller", func() {

	var (
		templateRecon *HypershiftTemplateReconciler
		err           error
		cancelContext context.CancelFunc
		ctx           context.Context

		k8sManager manager.Manager
	)

	BeforeEach(func() {
		k8sManager = getK8sManager(cfg)

		templateRecon = &HypershiftTemplateReconciler{
			Client:                 k8sManager.GetClient(),
			Scheme:                 k8sManager.GetScheme(),
			CreateDefaultTemplates: true,
		}
		err = templateRecon.SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		ctx, cancelContext = context.WithCancel(context.TODO())
		go func() {
			err = k8sManager.Start(ctx)
			Expect(err).ToNot(HaveOccurred())
		}()
	})

	AfterEach(func() {
		cancelContext()
	})

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
			ct.Spec.Cost = 500
			err = k8sClient.Update(ctx, ct)
			Expect(err).Should(BeNil())
			Eventually(func() bool {
				err := k8sClient.Get(
					ctx,
					client.ObjectKeyFromObject(ct),
					ct,
				)
				Expect(err).Should(BeNil())
				return ct.Spec.Cost == 1
			}, timeout, interval).Should(BeTrue())
		}
	})
})

var _ = Describe("HypershiftTemplate controller default template", func() {

	var (
		templateRecon *HypershiftTemplateReconciler
		err           error
		cancelContext context.CancelFunc
		ctx           context.Context

		k8sManager manager.Manager
	)

	BeforeEach(func() {
		k8sManager = getK8sManager(cfg)

		templateRecon = &HypershiftTemplateReconciler{
			Client:                 k8sManager.GetClient(),
			Scheme:                 k8sManager.GetScheme(),
			CreateDefaultTemplates: false,
		}
		err = templateRecon.SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		ctx, cancelContext = context.WithCancel(context.TODO())
		go func() {
			err = k8sManager.Start(ctx)
			Expect(err).ToNot(HaveOccurred())
		}()
	})

	AfterEach(func() {
		cancelContext()
	})

	It("Don't create default templates", func() {
		for template := range defaultTemplates {
			ct := &v1alpha1.ClusterTemplate{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: template}, ct)
			Expect(err).Should(HaveOccurred())
		}
	})
})
