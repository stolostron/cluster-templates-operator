package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"

	console "github.com/openshift/api/console/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

var _ = Describe("ConsolePlugin controller", func() {
	It("Creates all deployment resources", func() {
		Eventually(func() error {
			consolePlugin := getConsolePlugin()
			cp := &console.ConsolePlugin{}
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(consolePlugin), cp)
		}, timeout, interval).Should(BeNil())

		Eventually(func() error {
			pluginDeployment := getPluginDeployment()
			deployment := &appsv1.Deployment{}
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(pluginDeployment), deployment)
		}, timeout, interval).Should(BeNil())

		Eventually(func() error {
			pluginService := getPluginService()
			service := &v1.Service{}
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(pluginService), service)
		}, timeout, interval).Should(BeNil())

		Eventually(func() error {
			pluginCM := getPluginCM()
			cm := &v1.ConfigMap{}
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(pluginCM), cm)
		}, timeout, interval).Should(BeNil())
	})

	It("Recreates deployment", func() {
		pluginDeployment := getPluginDeployment()
		deployment := &appsv1.Deployment{}
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(pluginDeployment), deployment)
		Expect(err).Should(BeNil())
		deployment.Spec.Replicas = pointer.Int32(420)
		err = k8sClient.Update(ctx, deployment)
		Expect(err).Should(BeNil())
		Eventually(func() bool {
			err := k8sClient.Get(
				ctx,
				client.ObjectKeyFromObject(pluginDeployment),
				deployment,
			)
			Expect(err).Should(BeNil())
			return *deployment.Spec.Replicas == *pluginDeployment.Spec.Replicas
		}, timeout, interval).Should(BeTrue())
	})
})
