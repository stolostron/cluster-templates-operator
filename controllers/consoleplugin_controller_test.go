package controllers

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	applicationset "github.com/argoproj/applicationset/pkg/utils"
	console "github.com/openshift/api/console/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

var _ = Describe("ConsolePlugin controller", func() {
	AfterEach(func() {
		EnableUI = "false"
	})
	It("Creates all deployment resources", func() {
		EnableUI = "true"
		EnableUIconfigSync <- event.GenericEvent{Object: GetPluginDeployment()}
		Eventually(func() error {
			consolePlugin := getConsolePlugin()
			cp := &console.ConsolePlugin{}
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(consolePlugin), cp)
		}, timeout, interval).Should(BeNil())

		Eventually(func() error {
			pluginDeployment := GetPluginDeployment()
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
		EnableUI = "true"
		pluginDeployment := GetPluginDeployment()
		EnableUIconfigSync <- event.GenericEvent{Object: pluginDeployment}
		deployment := &appsv1.Deployment{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(pluginDeployment), deployment)
		}, timeout, interval).Should(BeNil())

		newDeployment := GetPluginDeployment()
		newDeployment.Spec.Replicas = pointer.Int32(420)
		_, err := applicationset.CreateOrUpdate(ctx, k8sClient, deployment, func() error {
			if !reflect.DeepEqual(deployment.Spec, newDeployment.Spec) {
				deployment.Spec = newDeployment.Spec
			}
			return nil
		})
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
