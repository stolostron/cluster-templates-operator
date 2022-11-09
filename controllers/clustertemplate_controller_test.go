package controllers

import (
	"net/http/httptest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	helmserver "github.com/stolostron/cluster-templates-operator/testutils/helm"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ClusterTemplate controller", func() {
	var server *httptest.Server
	ct := &v1alpha1.ClusterTemplate{}

	BeforeEach(func() {
		server = helmserver.StartHelmRepoServer()
		ct = &v1alpha1.ClusterTemplate{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.GroupVersion.Identifier(),
				Kind:       "ClusterTemplate",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: v1alpha1.ClusterTemplateSpec{
				ClusterDefinition: argo.ApplicationSpec{
					Source: argo.ApplicationSource{},
				},
			},
		}
	})

	AfterEach(func() {
		DeleteResource(ct)
		server.Close()
	})

	It("Should keep empty status when source is not Helm chart", func() {
		Expect(k8sClient.Create(ctx, ct)).Should(Succeed())
		Expect(ct.Status).Should(Equal(v1alpha1.ClusterTemplateStatus{}))
	})

	It("Should keep empty status when there are no values/schema", func() {
		ct.Spec.ClusterDefinition.Source.Chart = "hypershift-template-no-val-schema"
		ct.Spec.ClusterDefinition.Source.RepoURL = server.URL
		ct.Spec.ClusterDefinition.Source.TargetRevision = "0.0.2"
		Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

		Eventually(func() bool {
			foundCT := &v1alpha1.ClusterTemplate{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ct), foundCT)
			if err != nil {
				return false
			}

			return len(foundCT.Status.ClusterDefinition.Values) == 0 && len(foundCT.Status.ClusterDefinition.Schema) == 0
		}, timeout, interval).Should(BeTrue())
	})

	It("Should show values in status", func() {
		ct.Spec.ClusterDefinition.Source.Chart = "hypershift-template-no-schema"
		ct.Spec.ClusterDefinition.Source.RepoURL = server.URL
		ct.Spec.ClusterDefinition.Source.TargetRevision = "0.0.2"
		Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

		Eventually(func() bool {
			foundCT := &v1alpha1.ClusterTemplate{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ct), foundCT)
			if err != nil {
				return false
			}

			return len(foundCT.Status.ClusterDefinition.Values) > 0 && len(foundCT.Status.ClusterDefinition.Schema) == 0
		}, timeout, interval).Should(BeTrue())
	})

	It("Should show schema in status", func() {
		ct.Spec.ClusterDefinition.Source.Chart = "hypershift-template-no-val"
		ct.Spec.ClusterDefinition.Source.RepoURL = server.URL
		ct.Spec.ClusterDefinition.Source.TargetRevision = "0.0.2"
		Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

		Eventually(func() bool {
			foundCT := &v1alpha1.ClusterTemplate{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ct), foundCT)
			if err != nil {
				return false
			}

			return len(foundCT.Status.ClusterDefinition.Values) == 0 && len(foundCT.Status.ClusterDefinition.Schema) > 0
		}, timeout, interval).Should(BeTrue())
	})

	It("Should show values and schema in status", func() {
		ct.Spec.ClusterDefinition.Source.Chart = "hypershift-template"
		ct.Spec.ClusterDefinition.Source.RepoURL = server.URL
		ct.Spec.ClusterDefinition.Source.TargetRevision = "0.0.2"
		Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

		Eventually(func() bool {
			foundCT := &v1alpha1.ClusterTemplate{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ct), foundCT)
			if err != nil {
				return false
			}

			return len(foundCT.Status.ClusterDefinition.Values) > 0 && len(foundCT.Status.ClusterDefinition.Schema) > 0
		}, timeout, interval).Should(BeTrue())
	})

})
