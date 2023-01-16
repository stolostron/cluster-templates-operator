package controllers

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	applicationset "github.com/argoproj/applicationset/pkg/utils"
	testutils "github.com/stolostron/cluster-templates-operator/testutils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createResource(obj client.Object) {
	err := k8sClient.Create(ctx, obj)
	Expect(err).ToNot(HaveOccurred())
	resourcesToDelete = append(resourcesToDelete, obj)
}

var resourcesToDelete []client.Object

var _ = Describe("CLaaS Config", func() {
	AfterEach(func() {
		for _, res := range resourcesToDelete {
			testutils.DeleteResource(ctx, res, k8sClient)
		}
		resourcesToDelete = []client.Object{}
	})
	It("Updates argocd namespace", func() {
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claas-config",
				Namespace: "cluster-aas-operator",
			},
			Data: map[string]string{
				argoCDNsConfig: "default",
			},
		}
		createResource(cm)

		Eventually(func() bool {
			return ArgoCDNamespace == "default"
		}, timeout, interval).Should(BeTrue())
	})
	It("Enables UI deployment", func() {
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claas-config",
				Namespace: "cluster-aas-operator",
			},
			Data: map[string]string{
				enableUIConfig: "true",
			},
		}
		createResource(cm)

		deployment := &appsv1.Deployment{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(GetPluginDeployment()), deployment)
		}, timeout, interval).Should(BeNil())
		Expect(deployment.Spec.Template.Spec.Containers[0].Image == defaultUIImage)

		customImg := "quay.io/stolostron/foo:bar"
		updatedCm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claas-config",
				Namespace: "cluster-aas-operator",
			},
			Data: map[string]string{
				enableUIConfig: "true",
				uiImageConfig:  customImg,
			},
		}
		cm = &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claas-config",
				Namespace: "cluster-aas-operator",
			},
		}
		_, err := applicationset.CreateOrUpdate(ctx, k8sClient, cm, func() error {
			if !reflect.DeepEqual(cm.Data, updatedCm.Data) {
				cm.Data = updatedCm.Data
			}
			return nil
		})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(GetPluginDeployment()), deployment)
		}, timeout, interval).Should(BeNil())
		Expect(deployment.Spec.Template.Spec.Containers[0].Image == customImg)
	})
})
