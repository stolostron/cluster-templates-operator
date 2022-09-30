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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	openshiftAPI "github.com/openshift/api/helm/v1beta1"
	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	"github.com/rawagner/cluster-templates-operator/api/v1alpha1"
	"github.com/rawagner/cluster-templates-operator/helm"
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

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
			filepath.Join("..", "testutils", "testcrds"),
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
	err = openshiftAPI.Install(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = hypershiftv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = pipeline.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	helmClient := CreateHelmClient(k8sManager, cfg)
	err = (&ClusterTemplateInstanceReconciler{
		Client:         k8sManager.GetClient(),
		Scheme:         k8sManager.GetScheme(),
		HelmClient:     helmClient,
		RequeueTimeout: 1 * time.Second,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&ClusterTemplateQuotaReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

}, 60)

var certDataFileName string
var keyDataFileName string
var caDataFileName string

var _ = AfterSuite(func() {
	cancel()
	defer os.Remove(certDataFileName)
	defer os.Remove(keyDataFileName)
	defer os.Remove(caDataFileName)
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func CreateHelmClient(k8sManager manager.Manager, config *rest.Config) *helm.HelmClient {
	certDataFile, err := os.CreateTemp("", "certdata-*")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer certDataFile.Close()

	err = ioutil.WriteFile(certDataFile.Name(), config.CertData, 0644)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	certDataFileName = certDataFile.Name()

	keyDataFile, err := os.CreateTemp("", "keydata-*")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer keyDataFile.Close()

	err = ioutil.WriteFile(keyDataFile.Name(), config.KeyData, 0644)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	keyDataFileName = keyDataFile.Name()

	caDataFile, err := os.CreateTemp("", "cadata-*")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer caDataFile.Close()

	err = ioutil.WriteFile(caDataFile.Name(), config.CAData, 0644)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	caDataFileName = caDataFile.Name()

	return helm.NewHelmClient(cfg, k8sManager.GetClient(), &certDataFileName, &keyDataFileName, &caDataFileName)
}
