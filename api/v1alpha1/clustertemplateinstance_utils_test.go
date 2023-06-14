package v1alpha1

import (
	"context"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/kubernetes-client/go-base/config/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var ctx context.Context
var cancel context.CancelFunc

var _ = Describe("ClusterTemplateInstance utils", func() {
	BeforeSuite(func() {
		ctx, cancel = context.WithCancel(context.TODO())
		err := AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())
		err = argo.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		err = admissionv1beta1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterSuite(func() {
		cancel()
	})
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
		appset := &argo.ApplicationSet{}

		params, err := cti.GetHelmParameters(appset, false)

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

		params, err = cti.GetHelmParameters(appset, false)

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
		appset = &argo.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "appset1",
			},
			Spec: argo.ApplicationSetSpec{
				Template: argo.ApplicationSetTemplate{
					Spec: argo.ApplicationSpec{
						Source: argo.ApplicationSource{
							Helm: &argo.ApplicationSourceHelm{
								Parameters: []argo.HelmParameter{
									{Name: "foo", Value: "baz"},
								},
							},
						},
					},
				},
			},
		}

		params, err = cti.GetHelmParameters(appset, false)

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
						Name:           "foo",
						Value:          "bar",
						ApplicationSet: "appset1",
					},
				},
			},
		}
		appset = &argo.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "appset1",
			},
			Spec: argo.ApplicationSetSpec{
				Template: argo.ApplicationSetTemplate{
					Spec: argo.ApplicationSpec{
						Source: argo.ApplicationSource{
							Helm: &argo.ApplicationSourceHelm{
								Parameters: []argo.HelmParameter{
									{Name: "foo1", Value: "baz"},
								},
							},
						},
					},
				},
			},
		}

		params, err = cti.GetHelmParameters(appset, false)

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
		appset := &argo.ApplicationSet{}

		params, err := cti.GetHelmParameters(appset, true)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(params).Should(Equal([]argo.HelmParameter{}))

		cti = ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				Parameters: []Parameter{
					{
						Name:           "foo",
						Value:          "bar",
						ApplicationSet: "foo-day2",
					},
				},
			},
		}
		appset = &argo.ApplicationSet{
			Spec: argo.ApplicationSetSpec{
				Template: argo.ApplicationSetTemplate{
					Spec: argo.ApplicationSpec{
						Source: argo.ApplicationSource{
							Helm: &argo.ApplicationSourceHelm{
								Parameters: []argo.HelmParameter{
									{Name: "foo", Value: "baz"},
								},
							},
						},
					},
				},
			},
		}
		params, err = cti.GetHelmParameters(appset, true)

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
		appset = &argo.ApplicationSet{
			Spec: argo.ApplicationSetSpec{
				Template: argo.ApplicationSetTemplate{
					Spec: argo.ApplicationSpec{
						Source: argo.ApplicationSource{
							Helm: &argo.ApplicationSourceHelm{
								Parameters: []argo.HelmParameter{
									{Name: "foo1", Value: "baz"},
								},
							},
						},
					},
				},
			},
		}
		params, err = cti.GetHelmParameters(appset, true)

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
				Namespace: "cluster-aas-operator",
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

		app, err := cti.GetDay1Application(ctx, client, "cluster-aas-operator")

		Expect(err).ShouldNot(HaveOccurred())
		Expect(app).ShouldNot(BeNil())

		clientWithoutApps := fake.NewFakeClientWithScheme(scheme.Scheme)

		app, err = cti.GetDay1Application(ctx, clientWithoutApps, "cluster-aas-operator")

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
		appset := &argo.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "cluster-aas-operator",
			},
			Spec: argo.ApplicationSetSpec{},
		}

		client := fake.NewFakeClientWithScheme(scheme.Scheme, appset)
		err := cti.CreateDay1Application(ctx, client, "cluster-aas-operator", false, "foo")

		Expect(err).ShouldNot(HaveOccurred())

		a := argo.ApplicationSetList{}
		Expect(client.List(ctx, &a)).Should(Succeed())

		Expect(len(a.Items[0].Spec.Generators)).To(Equal(1))
		Expect(len(a.Items[0].Spec.Generators[0].List.Elements)).To(Equal(1))

		s, err := a.Items[0].Spec.Generators[0].List.Elements[0].MarshalJSON()
		Expect(err).ToNot(HaveOccurred())
		Expect(string(s)).To(ContainSubstring("{\"instance_ns\":\"default\",\"url\":\"https://kubernetes.default.svc\"}"))
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
						Name:  "fooParam",
						Value: "foo",
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
		appset := argo.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "cluster-aas-operator",
			},
			Spec: argo.ApplicationSetSpec{
				Generators: []argo.ApplicationSetGenerator{{}},
				Template: argo.ApplicationSetTemplate{
					Spec: argo.ApplicationSpec{
						Source: argo.ApplicationSource{},
					},
				},
			},
		}

		client := fake.NewFakeClientWithScheme(scheme.Scheme, &kubeconfigSecret, &appset)
		err = cti.CreateDay2Applications(ctx, client, "cluster-aas-operator", []string{"foo"})
		Expect(err).ShouldNot(HaveOccurred())

		appsets := argo.ApplicationSetList{}
		Expect(client.List(ctx, &appsets)).Should(Succeed())

		//Expect(appsets.Items[0].Spec.Generators[0].String()).To(Equal(""))
		Expect(len(appsets.Items[0].Spec.Generators)).To(Equal(2))
		Expect(len(appsets.Items[0].Spec.Generators[1].List.Elements)).To(Equal(1))

		data, err = appsets.Items[0].Spec.Generators[1].List.Elements[0].MarshalJSON()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(data).To(ContainSubstring("foo-server"))
	})

	It(
		"GetSubjectsWithClusterTemplateUserRole, CreateDynamicRole and CreateDynamicRoleBinding",
		func() {
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

			objs := []runtime.Object{
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-role-binding-1",
						Namespace: cti.Namespace,
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.SchemeGroupVersion.Group,
						Kind:     "ClusterRole",
						Name:     "cluster-templates-user",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind: "User",
							Name: "test-user-1",
						},
						{
							Kind: "Group",
							Name: "test-group-1",
						},
						{
							Kind: "User",
							Name: "test-user-2",
						},
					},
				},

				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ignored-test-role-binding-2",
						Namespace: cti.Namespace,
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.SchemeGroupVersion.Group,
						Kind:     "ClusterRole",
						Name:     "non-cluster-templates-user",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind: "User",
							Name: "test-user-3",
						},
					},
				},

				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-role-binding-3",
						Namespace: cti.Namespace,
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.SchemeGroupVersion.Group,
						Kind:     "ClusterRole",
						Name:     "cluster-templates-user",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind: "User",
							Name: "test-user-4",
						},
					},
				},
			}

			client := fake.NewFakeClientWithScheme(scheme.Scheme, objs...)

			roleSubjects, err := cti.GetSubjectsWithClusterTemplateUserRole(ctx, client)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(roleSubjects).ShouldNot(BeNil())
			Expect(len(roleSubjects)).Should(Equal(4))

			role, err := cti.CreateDynamicRole(ctx, client)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(role).ShouldNot(BeNil())

			rb, err := cti.CreateDynamicRoleBinding(ctx, client, role, roleSubjects)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(rb).ShouldNot(BeNil())
		},
	)

	It("DeleteDay1Application - handles missing day1 app set", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: ClusterTemplateInstanceSpec{},
		}
		client := fake.NewFakeClientWithScheme(scheme.Scheme)
		err := cti.DeleteDay1Application(ctx, client, "default", "foo")
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("DeleteDay2Application - handles missing day2 app set", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: ClusterTemplateInstanceSpec{},
		}
		client := fake.NewFakeClientWithScheme(scheme.Scheme)
		err := cti.DeleteDay2Application(ctx, client, "default", []string{"foo", "bar"})
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Label destionation namespace - same as CTI", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: ClusterTemplateInstanceSpec{},
		}
		appset := &argo.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "baz",
				Namespace: "cluster-aas-operator",
			},
			Spec: argo.ApplicationSetSpec{
				Generators: []argo.ApplicationSetGenerator{{}},
				Template: argo.ApplicationSetTemplate{
					Spec: argo.ApplicationSpec{
						Source: argo.ApplicationSource{},
						Destination: argo.ApplicationDestination{
							Namespace: "{{ instance_ns }}",
						},
					},
				},
			},
		}
		defns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: cti.Namespace,
			},
		}
		client := fake.NewFakeClientWithScheme(scheme.Scheme, defns)
		cti.labelDestionationNamespace(ctx, appset, client, "argocdns")
		ns := &corev1.Namespace{}
		err := client.Get(
			ctx,
			types.NamespacedName{Name: cti.Namespace},
			ns,
		)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ns.Labels["argocd.argoproj.io/managed-by"]).Should(Equal("argocdns"))
	})
})
