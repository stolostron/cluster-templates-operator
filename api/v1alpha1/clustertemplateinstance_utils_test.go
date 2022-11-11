package v1alpha1

import (
	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/kubernetes-client/go-base/config/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("ClusterTemplateInstance utils", func() {
	It("GetKubeadminPassRef", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		}
		Expect(cti.GetKubeadminPassRef()).Should(Equal("foo-admin-password"))
	})
	It("GetKubeconfigRef", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		}
		Expect(cti.GetKubeconfigRef()).Should(Equal("foo-admin-kubeconfig"))
	})
	It("GetOwnerReference", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
				UID:  "foo-uid",
			},
		}
		Expect(cti.GetOwnerReference()).Should(Equal(metav1.OwnerReference{
			APIVersion: "clustertemplate.openshift.io/v1alpha1",
			Kind:       "ClusterTemplateInstance",
			Name:       "foo",
			UID:        "foo-uid",
		}))
	})
	It("GetHelmParameters day1", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		}

		ct := ClusterTemplate{
			Spec: ClusterTemplateSpec{
				ClusterDefinition: argo.ApplicationSpec{
					Source: argo.ApplicationSource{},
				},
			},
		}

		params, err := cti.GetHelmParameters(ct, "")

		Expect(err).ShouldNot(HaveOccurred())
		Expect(params).Should(Equal([]argo.HelmParameter{}))

		cti = ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				Parameters: []Parameter{
					{
						Name:  "foo",
						Value: "bar",
					},
				},
			},
		}

		params, err = cti.GetHelmParameters(ct, "")

		Expect(err).ShouldNot(HaveOccurred())
		Expect(params).Should(Equal([]argo.HelmParameter{
			{
				Name:  "foo",
				Value: "bar",
			},
		}))

		cti = ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				Parameters: []Parameter{
					{
						Name:  "foo",
						Value: "bar",
					},
				},
			},
		}

		ct = ClusterTemplate{
			Spec: ClusterTemplateSpec{
				ClusterDefinition: argo.ApplicationSpec{
					Source: argo.ApplicationSource{
						Helm: &argo.ApplicationSourceHelm{
							Parameters: []argo.HelmParameter{
								{
									Name:  "foo",
									Value: "baz",
								},
							},
						},
					},
				},
			},
		}

		params, err = cti.GetHelmParameters(ct, "")

		Expect(err).ShouldNot(HaveOccurred())
		Expect(params).Should(Equal([]argo.HelmParameter{
			{
				Name:  "foo",
				Value: "baz",
			},
		}))

		cti = ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				Parameters: []Parameter{
					{
						Name:  "foo",
						Value: "bar",
					},
				},
			},
		}

		ct = ClusterTemplate{
			Spec: ClusterTemplateSpec{
				ClusterDefinition: argo.ApplicationSpec{
					Source: argo.ApplicationSource{
						Helm: &argo.ApplicationSourceHelm{
							Parameters: []argo.HelmParameter{
								{
									Name:  "foo1",
									Value: "baz",
								},
							},
						},
					},
				},
			},
		}

		params, err = cti.GetHelmParameters(ct, "")

		Expect(err).ShouldNot(HaveOccurred())
		Expect(params).Should(Equal([]argo.HelmParameter{
			{
				Name:  "foo1",
				Value: "baz",
			},
			{
				Name:  "foo",
				Value: "bar",
			},
		}))
	})

	It("GetHelmParameters day2", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		}

		ct := ClusterTemplate{
			Spec: ClusterTemplateSpec{
				ClusterSetup: []ClusterSetup{
					{
						Name: "foo-day2",
						Spec: argo.ApplicationSpec{
							Source: argo.ApplicationSource{},
						},
					},
				},
			},
		}

		params, err := cti.GetHelmParameters(ct, "foo-day2")

		Expect(err).ShouldNot(HaveOccurred())
		Expect(params).Should(Equal([]argo.HelmParameter{}))

		cti = ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				Parameters: []Parameter{
					{
						Name:         "foo",
						Value:        "bar",
						ClusterSetup: "foo-day2",
					},
				},
			},
		}

		ct = ClusterTemplate{
			Spec: ClusterTemplateSpec{
				ClusterSetup: []ClusterSetup{
					{
						Name: "foo-day2",
						Spec: argo.ApplicationSpec{
							Source: argo.ApplicationSource{
								Helm: &argo.ApplicationSourceHelm{
									Parameters: []argo.HelmParameter{
										{
											Name:  "foo",
											Value: "baz",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		params, err = cti.GetHelmParameters(ct, "foo-day2")

		Expect(err).ShouldNot(HaveOccurred())
		Expect(params).Should(Equal([]argo.HelmParameter{
			{
				Name:  "foo",
				Value: "baz",
			},
		}))

		cti = ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				Parameters: []Parameter{
					{
						Name:         "foo",
						Value:        "bar",
						ClusterSetup: "foo-day2",
					},
				},
			},
		}

		ct = ClusterTemplate{
			Spec: ClusterTemplateSpec{
				ClusterSetup: []ClusterSetup{
					{
						Name: "foo-day2",
						Spec: argo.ApplicationSpec{
							Source: argo.ApplicationSource{
								Helm: &argo.ApplicationSourceHelm{
									Parameters: []argo.HelmParameter{
										{
											Name:  "foo1",
											Value: "baz",
										},
									},
								},
							},
						},
					},
				},
			},
		}
		params, err = cti.GetHelmParameters(ct, "foo-day2")

		Expect(err).ShouldNot(HaveOccurred())
		Expect(params).Should(Equal([]argo.HelmParameter{
			{
				Name:  "foo1",
				Value: "baz",
			},
			{
				Name:  "foo",
				Value: "bar",
			},
		}))
	})

	It("GetDay1Application", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: ClusterTemplateInstanceSpec{
				Parameters: []Parameter{
					{
						Name:  "foo",
						Value: "bar",
					},
				},
			},
		}

		argoApp := &argo.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-bar",
				Namespace: ArgoNamespace,
				Labels: map[string]string{
					CTINameLabel:      "foo",
					CTINamespaceLabel: "default",
				},
			},
			Spec: argo.ApplicationSpec{
				Source:      argo.ApplicationSource{},
				Destination: argo.ApplicationDestination{},
				Project:     "",
			},
		}

		client := fake.NewFakeClientWithScheme(scheme.Scheme, argoApp)

		app, err := cti.GetDay1Application(ctx, client)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(app).ShouldNot(BeNil())

		clientWithoutApps := fake.NewFakeClientWithScheme(scheme.Scheme)

		app, err = cti.GetDay1Application(ctx, clientWithoutApps)

		Expect(err).Should(HaveOccurred())
		Expect(app).Should(BeNil())
	})

	It("CreateDay1Application", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: ClusterTemplateInstanceSpec{
				Parameters: []Parameter{
					{
						Name:  "fooParam",
						Value: "foo",
					},
				},
			},
		}

		ct := ClusterTemplate{
			Spec: ClusterTemplateSpec{
				ClusterDefinition: argo.ApplicationSpec{
					Source: argo.ApplicationSource{
						RepoURL: "http://foo",
					},
				},
			},
		}

		client := fake.NewFakeClientWithScheme(scheme.Scheme)
		err := cti.CreateDay1Application(ctx, client, ct)

		Expect(err).ShouldNot(HaveOccurred())

		apps := argo.ApplicationList{}
		Expect(client.List(ctx, &apps)).Should(Succeed())

		Expect(apps.Items[0].Namespace).To(Equal(ArgoNamespace))
		Expect(apps.Items[0].Labels[CTINameLabel]).To(Equal("foo"))
		Expect(apps.Items[0].Labels[CTINamespaceLabel]).To(Equal("default"))

		ct = ClusterTemplate{
			Spec: ClusterTemplateSpec{
				ClusterDefinition: argo.ApplicationSpec{
					Source: argo.ApplicationSource{
						RepoURL: "http://foo",
					},
					Destination: argo.ApplicationDestination{
						Namespace: CTIInstanceNamespaceVar,
					},
				},
			},
		}

		client = fake.NewFakeClientWithScheme(scheme.Scheme)
		err = cti.CreateDay1Application(ctx, client, ct)

		Expect(err).ShouldNot(HaveOccurred())

		apps = argo.ApplicationList{}
		Expect(client.List(ctx, &apps)).Should(Succeed())

		Expect(apps.Items[0].Namespace).To(Equal(ArgoNamespace))
		Expect(apps.Items[0].Labels[CTINameLabel]).To(Equal("foo"))
		Expect(apps.Items[0].Labels[CTINamespaceLabel]).To(Equal("default"))
		Expect(apps.Items[0].Spec.Destination.Namespace).To(Equal("default"))
		Expect(apps.Items[0].Spec.Source.Helm.Parameters[0].Name).To(Equal("fooParam"))
		Expect(apps.Items[0].Spec.Source.Helm.Parameters[0].Value).To(Equal("foo"))

	})

	It("CreateDay2Applications", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: ClusterTemplateInstanceSpec{
				Parameters: []Parameter{
					{
						Name:         "fooParam",
						Value:        "foo",
						ClusterSetup: "foo-day2",
					},
				},
			},
		}

		ct := ClusterTemplate{
			Spec: ClusterTemplateSpec{
				ClusterSetup: []ClusterSetup{
					{
						Name: "foo-day2",
						Spec: argo.ApplicationSpec{
							Source: argo.ApplicationSource{
								RepoURL: "http://foo",
							},
						},
					},
				},
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
		Expect(err).ShouldNot(HaveOccurred())
		kubeconfigSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cti.GetKubeconfigRef(),
				Namespace: cti.Namespace,
			},
			Data: map[string][]byte{
				"kubeconfig": data,
			},
		}

		client := fake.NewFakeClientWithScheme(scheme.Scheme, &kubeconfigSecret)
		err = cti.CreateDay2Applications(ctx, client, ct)
		Expect(err).ShouldNot(HaveOccurred())

		apps := argo.ApplicationList{}
		Expect(client.List(ctx, &apps)).Should(Succeed())

		Expect(apps.Items[0].Namespace).To(Equal(ArgoNamespace))
		Expect(apps.Items[0].Labels[CTINameLabel]).To(Equal("foo"))
		Expect(apps.Items[0].Labels[CTINamespaceLabel]).To(Equal("default"))
		Expect(apps.Items[0].Labels[CTISetupLabel]).To(Equal("foo-day2"))
		Expect(apps.Items[0].Spec.Source.Helm.Parameters[0].Name).To(Equal("fooParam"))
		Expect(apps.Items[0].Spec.Source.Helm.Parameters[0].Value).To(Equal("foo"))

		ct = ClusterTemplate{
			Spec: ClusterTemplateSpec{
				ClusterSetup: []ClusterSetup{
					{
						Name: "foo-day2",
						Spec: argo.ApplicationSpec{
							Source: argo.ApplicationSource{
								RepoURL: "http://foo",
							},
							Destination: argo.ApplicationDestination{
								Server: CTIClusterTargetVar,
							},
						},
					},
				},
			},
		}

		client = fake.NewFakeClientWithScheme(scheme.Scheme, &kubeconfigSecret)
		err = cti.CreateDay2Applications(ctx, client, ct)
		Expect(err).ShouldNot(HaveOccurred())

		apps = argo.ApplicationList{}
		Expect(client.List(ctx, &apps)).Should(Succeed())

		Expect(apps.Items[0].Namespace).To(Equal(ArgoNamespace))
		Expect(apps.Items[0].Labels[CTINameLabel]).To(Equal("foo"))
		Expect(apps.Items[0].Labels[CTINamespaceLabel]).To(Equal("default"))
		Expect(apps.Items[0].Labels[CTISetupLabel]).To(Equal("foo-day2"))
		Expect(apps.Items[0].Spec.Destination.Server).To(Equal("foo-server"))
	})
})
