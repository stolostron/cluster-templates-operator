package controllers

import (
	"net/http/httptest"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	testutils "github.com/stolostron/cluster-templates-operator/testutils"
	helmserver "github.com/stolostron/cluster-templates-operator/testutils/helm"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ClusterTemplate controller", func() {
	var server *httptest.Server
	ct := &v1alpha1.ClusterTemplate{}
	var appset *argo.ApplicationSet

	BeforeEach(func() {
		server = helmserver.StartHelmRepoServer()
		ct = &v1alpha1.ClusterTemplate{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.GroupVersion.Identifier(),
				Kind:       "ClusterTemplate",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "argocd",
			},
			Spec: v1alpha1.ClusterTemplateSpec{
				ClusterDefinition: "foo",
			},
		}
		appset = &argo.ApplicationSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       argo.ApplicationSetSchemaGroupVersionKind.Kind,
				APIVersion: argo.ApplicationSetSchemaGroupVersionKind.GroupVersion().Identifier(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "argocd",
			},
			Spec: argo.ApplicationSetSpec{
				Generators: []argo.ApplicationSetGenerator{{}},
				Template: argo.ApplicationSetTemplate{
					Spec: argo.ApplicationSpec{
						Source: argo.ApplicationSource{
							RepoURL:        server.URL,
							TargetRevision: "0.0.2",
							Chart:          "hypershift-template-no-schema",
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, ct)).Should(Succeed())
	})

	AfterEach(func() {
		testutils.DeleteResource(ctx, ct, k8sClient)
		testutils.DeleteResource(ctx, appset, k8sClient)
		server.Close()
	})

	It("Should keep empty status when source is not Helm chart", func() {
		Expect(k8sClient.Create(ctx, appset)).Should(Succeed())
		Expect(ct.Status).Should(Equal(v1alpha1.ClusterTemplateStatus{}))
	})

	It("Should keep empty status when there are no values/schema", func() {
		appset.Spec.Template.Spec.Source.Chart = "hypershift-template-no-val-schema"
		Expect(k8sClient.Create(ctx, appset)).Should(Succeed())

		Eventually(func() bool {
			foundCT := &v1alpha1.ClusterTemplate{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ct), foundCT)
			if err != nil {
				return false
			}

			return len(foundCT.Status.ClusterDefinition.Values) == 0 &&
				len(foundCT.Status.ClusterDefinition.Schema) == 0
		}, timeout, interval).Should(BeTrue())
	})

	It("Should show values in status", func() {
		appset.Spec.Template.Spec.Source.Chart = "hypershift-template-no-schema"
		Expect(k8sClient.Create(ctx, appset)).Should(Succeed())

		Eventually(func() bool {
			foundCT := &v1alpha1.ClusterTemplate{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ct), foundCT)
			if err != nil {
				return false
			}

			return len(foundCT.Status.ClusterDefinition.Values) > 0 &&
				len(foundCT.Status.ClusterDefinition.Schema) == 0
		}, timeout, interval).Should(BeTrue())
	})

	It("Should show schema in status", func() {
		appset.Spec.Template.Spec.Source.Chart = "hypershift-template-no-val"
		Expect(k8sClient.Create(ctx, appset)).Should(Succeed())

		Eventually(func() bool {
			foundCT := &v1alpha1.ClusterTemplate{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ct), foundCT)
			if err != nil {
				return false
			}

			return len(foundCT.Status.ClusterDefinition.Values) == 0 &&
				len(foundCT.Status.ClusterDefinition.Schema) > 0
		}, timeout, interval).Should(BeTrue())
	})

	It("Should show values and schema in status", func() {
		appset.Spec.Template.Spec.Source.Chart = "hypershift-template"
		Expect(k8sClient.Create(ctx, appset)).Should(Succeed())

		Eventually(func() bool {
			foundCT := &v1alpha1.ClusterTemplate{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ct), foundCT)
			if err != nil {
				return false
			}

			return len(foundCT.Status.ClusterDefinition.Values) > 0 &&
				len(foundCT.Status.ClusterDefinition.Schema) > 0
		}, timeout, interval).Should(BeTrue())
	})

	It("Should set error for ClusterDefinition in case of invalid port", func() {
		appset.Spec.Template.Spec.Source.Chart = "hypershift-template"
		appset.Spec.Template.Spec.Source.RepoURL = server.URL + "NONEXISTING"
		Expect(k8sClient.Create(ctx, appset)).Should(Succeed())

		Eventually(func() bool {
			foundCT := &v1alpha1.ClusterTemplate{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ct), foundCT)
			if err != nil {
				return false
			}

			if foundCT.Status.ClusterDefinition.Error != nil {
				return strings.Contains(*foundCT.Status.ClusterDefinition.Error, "invalid port")
			}
			return false
		}, timeout, interval).Should(BeTrue())
	})

})
