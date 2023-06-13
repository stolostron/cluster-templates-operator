/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	argooperator "github.com/argoproj-labs/argocd-operator/api/v1alpha1"
	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	consoleV1 "github.com/openshift/api/console/v1"
	console "github.com/openshift/api/console/v1alpha1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/v1beta1"
	operators "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	testutils "github.com/stolostron/cluster-templates-operator/testutils"
	agent "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ocmv1 "open-cluster-management.io/api/cluster/v1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc
var controllerCancel context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join("..", "testutils", "testcrds", "required"),
			filepath.Join("..", "testutils", "testcrds", "optional", "hypershift"),
			filepath.Join("..", "testutils", "testcrds", "optional", "hive"),
			filepath.Join("..", "testutils", "testcrds", "optional", "console"),
			filepath.Join("..", "testutils", "testcrds", "optional", "consoleV1"),
		},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = hypershiftv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = hivev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = argo.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = console.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = ocmv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = argooperator.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = operators.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = consoleV1.AddToScheme(scheme.Scheme)
	err = agent.SchemeBuilder.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	controllerCancel = StartCTIController(k8sManager, true, false, false, false)

	claasNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: pluginNamespace,
		},
	}
	err = k8sManager.GetClient().Create(ctx, claasNs)
	Expect(err).ToNot(HaveOccurred())

	err = (&ConfigReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&ClusterTemplateQuotaReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&ClusterTemplateReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&ConsolePluginReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&ConfigReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&HypershiftTemplateReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&ArgoCDReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openshift-operators",
		},
	}
	nsArgo := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-argocd-ns",
		},
	}
	Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	Expect(k8sClient.Create(ctx, nsArgo)).Should(Succeed())
	Expect(k8sClient.Create(ctx, testutils.GetSubscription(nil))).Should(Succeed())
}, 60)

var certDataFileName string
var keyDataFileName string
var caDataFileName string

var _ = AfterSuite(func() {
	controllerCancel()
	cancel()
	defer os.Remove(certDataFileName)
	defer os.Remove(keyDataFileName)
	defer os.Remove(caDataFileName)
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

const (
	timeout  = time.Second * 10
	duration = time.Second * 10
	interval = time.Millisecond * 250
)
