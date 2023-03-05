package ocm

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	agent "github.com/stolostron/klusterlet-addon-controller/pkg/apis"
	agentv1 "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ocmv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Klusterlet", func() {
	scheme := runtime.NewScheme()
	utilruntime.Must(ocmv1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(agent.AddToScheme(scheme))
	It("Returns error when MC does not exist", func() {
		client := fake.NewFakeClientWithScheme(scheme)
		cti := &v1alpha1.ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Status: v1alpha1.ClusterTemplateInstanceStatus{
				ClusterTemplateLabels: map[string]string{
					"foo": "bar",
				},
			},
		}
		err := CreateKlusterletAddonConfig(context.TODO(), client, cti)
		Expect(err).Should(HaveOccurred())
	})
	It("Creates klusterlet when it does not exist", func() {
		cti := &v1alpha1.ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Status: v1alpha1.ClusterTemplateInstanceStatus{
				ClusterTemplateLabels: map[string]string{
					"foo": "bar",
				},
			},
		}
		mc := &ocmv1.ManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-foo",
				Labels: map[string]string{
					v1alpha1.CTINameLabel:      cti.Name,
					v1alpha1.CTINamespaceLabel: cti.Namespace,
				},
			},
			Spec: ocmv1.ManagedClusterSpec{
				HubAcceptsClient: true,
			},
		}
		client := fake.NewFakeClientWithScheme(scheme, mc)
		err := CreateKlusterletAddonConfig(context.TODO(), client, cti)
		Expect(err).ShouldNot(HaveOccurred())

		klusterlet := &agentv1.KlusterletAddonConfig{}
		err = client.Get(context.TODO(), types.NamespacedName{Name: mc.Name, Namespace: mc.Name}, klusterlet)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(klusterlet.Name).To(Equal(mc.Name))
	})
})
