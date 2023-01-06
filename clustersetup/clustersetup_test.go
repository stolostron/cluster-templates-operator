package clustersetup

import (
	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/kubernetes-client/go-base/config/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func GetNewClient(configBytes []byte) (client.Client, error) {
	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-manager" + "-token",
			Namespace: "kube-system",
		},
		Type: corev1.SecretTypeServiceAccountToken,
		Data: map[string][]byte{
			"token":  []byte("token"),
			"ca.crt": []byte("ca.crt"),
		},
	}
	client := fake.NewFakeClientWithScheme(scheme.Scheme, tokenSecret)
	return client, nil
}

var _ = Describe("Test cluster setup", func() {
	It("AddClusterToArgo", func() {
		cti := &v1alpha1.ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Status: v1alpha1.ClusterTemplateInstanceStatus{
				ClusterTemplateSpec: &v1alpha1.ClusterTemplateSpec{},
			},
		}

		kubeconfig := api.Config{
			Clusters: []api.NamedCluster{
				{
					Name: "foo",
					Cluster: api.Cluster{
						Server: "fooapi",
					},
				},
			},
		}
		data, err := yaml.Marshal(kubeconfig)

		Expect(err).Should(BeNil())

		kubeconfigSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cti.GetKubeconfigRef(),
				Namespace: cti.Namespace,
			},
			Data: map[string][]byte{
				"kubeconfig": data,
			},
		}

		app := &argo.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "argocd",
				Labels: map[string]string{
					v1alpha1.CTINameLabel:      cti.Name,
					v1alpha1.CTINamespaceLabel: cti.Namespace,
				},
			},
		}

		client := fake.NewFakeClientWithScheme(scheme.Scheme, kubeconfigSecret, app)
		err = AddClusterToArgo(ctx, client, cti, GetNewClient, "argocd")
		Expect(err).Should(BeNil())

		argoClusterSecret := &corev1.Secret{}
		err = client.Get(
			ctx,
			types.NamespacedName{Name: app.Name, Namespace: app.Namespace},
			argoClusterSecret,
		)
		Expect(err).Should(BeNil())
	})
})
