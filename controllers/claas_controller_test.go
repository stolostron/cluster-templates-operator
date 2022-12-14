package controllers

import (
	"context"
	"path/filepath"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var claasCtx context.Context
var claasCtxCancel context.CancelFunc
var claasTestEnv *envtest.Environment
var claasK8sManager manager.Manager
var claasK8sClient client.Client
var claasReconciler *CLaaSReconciler

var _ = Describe("CLaaS controller", func() {
	AfterEach(func() {
		claasCtxCancel()
		err := claasTestEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	})
	It("Can create CLaaS reconciler", func() {
		startTestEnv([]string{})
		Expect(claasReconciler.enableHypershift).Should(BeFalse())
		Expect(claasReconciler.enableHive).Should(BeFalse())
		Expect(claasReconciler.enableHelmRepo).Should(BeFalse())
		Expect(ctiControllerCancel).ShouldNot(BeNil())
	})
	It("CLaaS reconciler start - detects hive", func() {
		startTestEnv([]string{
			filepath.Join("..", "testutils", "testcrds", "optional", "hive"),
		})
		Expect(claasReconciler.enableHypershift).Should(BeFalse())
		Expect(claasReconciler.enableHive).Should(BeTrue())
		Expect(claasReconciler.enableHelmRepo).Should(BeFalse())
		Expect(ctiControllerCancel).ShouldNot(BeNil())
	})
	It("CLaaS reconciler start - detects hypershift", func() {
		startTestEnv([]string{
			filepath.Join("..", "testutils", "testcrds", "optional", "hypershift"),
		})
		Expect(claasReconciler.enableHypershift).Should(BeTrue())
		Expect(claasReconciler.enableHive).Should(BeFalse())
		Expect(claasReconciler.enableHelmRepo).Should(BeFalse())
		Expect(ctiControllerCancel).ShouldNot(BeNil())
	})
	It("CLaaS reconciler start - detects helm", func() {
		startTestEnv([]string{
			filepath.Join("..", "testutils", "testcrds", "optional", "helm"),
		})
		Expect(claasReconciler.enableHypershift).Should(BeFalse())
		Expect(claasReconciler.enableHive).Should(BeFalse())
		Expect(claasReconciler.enableHelmRepo).Should(BeTrue())
		Expect(ctiControllerCancel).ShouldNot(BeNil())
	})
	It("enables hypershift dynamically", func() {
		startTestEnv([]string{})
		Expect(claasReconciler.enableHypershift).Should(BeFalse())
		Expect(claasReconciler.enableHive).Should(BeFalse())
		Expect(claasReconciler.enableHelmRepo).Should(BeFalse())
		Expect(ctiControllerCancel).ShouldNot(BeNil())

		err := claasK8sClient.Create(claasCtx, &apiextensions.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "hostedclusters.hypershift.openshift.io",
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Scope: apiextensions.NamespaceScoped,
				Group: v1alpha1.HostedClusterGVK.Group,
				Versions: []apiextensions.CustomResourceDefinitionVersion{
					{
						Name:    v1alpha1.HostedClusterGVK.Version,
						Storage: true,
						Schema: &apiextensions.CustomResourceValidation{
							OpenAPIV3Schema: &apiextensions.JSONSchemaProps{
								Type: "object",
							},
						},
					},
				},
				Names: apiextensions.CustomResourceDefinitionNames{
					Kind:   v1alpha1.HostedClusterGVK.Resource,
					Plural: "hostedclusters",
				},
			},
		})
		Expect(err).Should(BeNil())

		Eventually(func() bool {
			return claasReconciler.enableHypershift
		}, timeout, interval).Should(BeTrue())
		Expect(claasReconciler.enableHive).Should(BeFalse())
		Expect(claasReconciler.enableHelmRepo).Should(BeFalse())
	})
	It("enables hive dynamically", func() {
		startTestEnv([]string{})
		Expect(claasReconciler.enableHypershift).Should(BeFalse())
		Expect(claasReconciler.enableHive).Should(BeFalse())
		Expect(claasReconciler.enableHelmRepo).Should(BeFalse())
		Expect(ctiControllerCancel).ShouldNot(BeNil())

		err := claasK8sClient.Create(claasCtx, &apiextensions.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clusterdeployments.hive.openshift.io",
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Scope: apiextensions.NamespaceScoped,
				Group: v1alpha1.ClusterDeploymentGVK.Group,
				Versions: []apiextensions.CustomResourceDefinitionVersion{
					{
						Name:    v1alpha1.ClusterDeploymentGVK.Version,
						Storage: true,
						Schema: &apiextensions.CustomResourceValidation{
							OpenAPIV3Schema: &apiextensions.JSONSchemaProps{
								Type: "object",
							},
						},
					},
				},
				Names: apiextensions.CustomResourceDefinitionNames{
					Kind:   v1alpha1.ClusterDeploymentGVK.Resource,
					Plural: "clusterdeployments",
				},
			},
		})
		Expect(err).Should(BeNil())

		Eventually(func() bool {
			return claasReconciler.enableHive
		}, timeout, interval).Should(BeTrue())
		Expect(claasReconciler.enableHypershift).Should(BeFalse())
		Expect(claasReconciler.enableHelmRepo).Should(BeFalse())
	})
	It("enables helm dynamically", func() {
		startTestEnv([]string{})
		Expect(claasReconciler.enableHypershift).Should(BeFalse())
		Expect(claasReconciler.enableHive).Should(BeFalse())
		Expect(claasReconciler.enableHelmRepo).Should(BeFalse())

		err := claasK8sClient.Create(claasCtx, &apiextensions.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "helmchartrepositories.helm.openshift.io",
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Scope: apiextensions.NamespaceScoped,
				Group: v1alpha1.HelmRepoGVK.Group,
				Versions: []apiextensions.CustomResourceDefinitionVersion{
					{
						Name:    v1alpha1.HelmRepoGVK.Version,
						Storage: true,
						Schema: &apiextensions.CustomResourceValidation{
							OpenAPIV3Schema: &apiextensions.JSONSchemaProps{
								Type: "object",
							},
						},
					},
				},
				Names: apiextensions.CustomResourceDefinitionNames{
					Kind:   v1alpha1.HelmRepoGVK.Resource,
					Plural: "helmchartrepositories",
				},
			},
		})
		Expect(err).Should(BeNil())

		Eventually(func() bool {
			return claasReconciler.enableHelmRepo
		}, timeout, interval).Should(BeTrue())
		Expect(claasReconciler.enableHive).Should(BeFalse())
		Expect(claasReconciler.enableHypershift).Should(BeFalse())
	})
})

func startTestEnv(crds []string) {
	claasCtx, claasCtxCancel = context.WithCancel(context.TODO())
	crdPaths := []string{
		filepath.Join("..", "config", "crd", "bases"),
		filepath.Join("..", "testutils", "testcrds", "required"),
	}
	crdPaths = append(crdPaths, crds...)
	claasTestEnv = &envtest.Environment{
		CRDDirectoryPaths:     crdPaths,
		ErrorIfCRDPathMissing: true,
	}
	// cfg is defined in this file globally.
	cfg, err := claasTestEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = hypershiftv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = hivev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = argo.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	claasK8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(claasK8sClient).NotTo(BeNil())

	claasK8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		Port:               9000,
		MetricsBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	go func() {
		//defer GinkgoRecover()
		err = claasK8sManager.Start(claasCtx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	claasReconciler = &CLaaSReconciler{
		Client:  claasK8sManager.GetClient(),
		Manager: claasK8sManager,
	}
	err = claasReconciler.SetupWithManager()
	Expect(err).ToNot(HaveOccurred())
}
