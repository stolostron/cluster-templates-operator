package ocm

import (
	"context"

	"github.com/kubernetes-client/go-base/config/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ocmv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("ManagedCluster", func() {
	scheme := runtime.NewScheme()
	utilruntime.Must(ocmv1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	It("Create managed cluster", func() {
		client := fake.NewFakeClientWithScheme(scheme)
		cti := &v1alpha1.ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				Labels: map[string]string{
					"foo": "bar",
				},
			},
		}
		CreateManagedCluster(context.TODO(), client, cti, map[string]string{"foo": "bar"})
		mcs := &ocmv1.ManagedClusterList{}
		client.List(context.TODO(), mcs)
		Expect(len(mcs.Items)).To(Equal(1))
		Expect(mcs.Items[0].Labels["foo"]).To(Equal("bar"))
	})

	It("Get managed cluster", func() {
		mc := &ocmv1.ManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-mc",
				Labels: map[string]string{
					v1alpha1.CTINamespaceLabel: "bar",
					v1alpha1.CTINameLabel:      "foo",
				},
			},
		}
		client := fake.NewFakeClientWithScheme(scheme, mc)
		cti := &v1alpha1.ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
		}
		foundMc, err := GetManagedCluster(context.TODO(), client, cti)
		Expect(err).To(BeNil())
		Expect(foundMc).To(Equal(mc))
	})

	It("Import managed cluster", func() {
		mc := &ocmv1.ManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-mc",
				Labels: map[string]string{
					v1alpha1.CTINamespaceLabel: "bar",
					v1alpha1.CTINameLabel:      "foo",
				},
			},
		}
		cti := &v1alpha1.ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
		}
		kubeconfig := api.Config{}
		kubeconfig.Clusters = []api.NamedCluster{
			{
				Name: "foo",
				Cluster: api.Cluster{
					Server: "foo-server",
				},
			},
		}
		data, err := yaml.Marshal(&kubeconfig)
		Expect(err).To(BeNil())
		kubeconfigSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cti.GetKubeconfigRef(),
				Namespace: cti.Namespace,
			},
			Data: map[string][]byte{
				"kubeconfig": data,
			},
		}
		client := fake.NewFakeClientWithScheme(scheme, mc, kubeconfigSecret)

		imported, err := ImportManagedCluster(context.TODO(), client, cti)
		Expect(err).To(BeNil())
		Expect(imported).To(BeFalse())

		importSecretMeta := GetImportSecretMeta(mc.Name)
		secret := &corev1.Secret{}
		err = client.Get(
			context.TODO(),
			types.NamespacedName{
				Namespace: importSecretMeta.Namespace,
				Name:      importSecretMeta.Name,
			},
			secret,
		)
		Expect(err).To(BeNil())
		Expect(secret.Name).NotTo(Equal(""))

	})
})
