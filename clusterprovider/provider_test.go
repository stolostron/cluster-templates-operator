package clusterprovider

import (
	"encoding/json"
	"os"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/kubernetes-client/go-base/config/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Test cluster providers", func() {
	err := hypershiftv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	cti := v1alpha1.ClusterTemplateInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cti",
			Namespace: "bar",
		},
	}

	hypershiftProvider := HostedClusterProvider{
		HostedClusterName:      "foo",
		HostedClusterNamespace: "bar",
	}
	Context("Test HostedCluster provider", func() {
		testProvider(hypershiftProvider, cti, getHostedCluster)
	})

	hypershiftProvider = HostedClusterProvider{
		HostedClusterName:      "foo",
		HostedClusterNamespace: "bar",
		NodePoolNames:          []string{"np1"},
	}
	Context("Test HostedCluster provider with nodepools", func() {
		testProvider(hypershiftProvider, cti, getHostedClusterWithNodePools)
	})

	clusterClaimProvider := ClusterClaimProvider{
		ClusterClaimName:      "foo",
		ClusterClaimNamespace: "bar",
	}
	Context("Test ClusterClaim provider", func() {
		testProvider(clusterClaimProvider, cti, getClusterClaim)
	})

	clusterDeploymentProvider := ClusterDeploymentProvider{
		ClusterDeploymentName:      "foo",
		ClusterDeploymentNamespace: "bar",
	}
	Context("Test ClusterDeployment provider", func() {
		testProvider(clusterDeploymentProvider, cti, getClusterDeployment)
	})

	Context("Detect cluster provider", func() {
		log := ctrl.LoggerFrom(ctx)
		app := argo.Application{
			Status: argo.ApplicationStatus{},
		}
		provider := GetClusterProvider(app, log)
		Expect(provider).Should(BeNil())

		app = argo.Application{
			Status: argo.ApplicationStatus{
				Resources: []argo.ResourceStatus{
					{
						Kind:      "HostedCluster",
						Version:   "v1alpha1",
						Name:      "foo",
						Namespace: "bar",
					},
				},
			},
		}
		provider = GetClusterProvider(app, log)
		expectedProvider := HostedClusterProvider{
			HostedClusterName:      "foo",
			HostedClusterNamespace: "bar",
			NodePoolNames:          []string{},
		}
		Expect(provider).Should(Equal(expectedProvider))

		app = argo.Application{
			Status: argo.ApplicationStatus{
				Resources: []argo.ResourceStatus{
					{
						Kind:      "HostedCluster",
						Version:   "v1alpha2",
						Name:      "foo",
						Namespace: "bar",
					},
				},
			},
		}
		provider = GetClusterProvider(app, log)
		Expect(provider).Should(BeNil())

		app = argo.Application{
			Status: argo.ApplicationStatus{
				Resources: []argo.ResourceStatus{
					{
						Kind:      "HostedCluster",
						Version:   "v1alpha1",
						Name:      "foo",
						Namespace: "bar",
					},
					{
						Kind:      "NodePool",
						Version:   "v1alpha1",
						Name:      "np-foo",
						Namespace: "bar",
					},
				},
			},
		}
		provider = GetClusterProvider(app, log)
		Expect(provider).Should(Equal(HostedClusterProvider{
			HostedClusterName:      "foo",
			HostedClusterNamespace: "bar",
			NodePoolNames:          []string{"np-foo"},
		}))

		app = argo.Application{
			Status: argo.ApplicationStatus{
				Resources: []argo.ResourceStatus{
					{
						Kind:      "ClusterDeployment",
						Version:   "v1",
						Name:      "foo",
						Namespace: "bar",
					},
				},
			},
		}
		provider = GetClusterProvider(app, log)
		Expect(provider).Should(Equal(ClusterDeploymentProvider{
			ClusterDeploymentName:      "foo",
			ClusterDeploymentNamespace: "bar",
		}))

		app = argo.Application{
			Status: argo.ApplicationStatus{
				Resources: []argo.ResourceStatus{
					{
						Kind:      "ClusterDeployment",
						Version:   "v1alpha1",
						Name:      "foo",
						Namespace: "bar",
					},
				},
			},
		}
		provider = GetClusterProvider(app, log)
		Expect(provider).Should(BeNil())

		app = argo.Application{
			Status: argo.ApplicationStatus{
				Resources: []argo.ResourceStatus{
					{
						Kind:      "ClusterClaim",
						Version:   "v1",
						Name:      "foo",
						Namespace: "bar",
					},
				},
			},
		}
		provider = GetClusterProvider(app, log)
		Expect(provider).Should(Equal(ClusterClaimProvider{
			ClusterClaimName:      "foo",
			ClusterClaimNamespace: "bar",
		}))

		app = argo.Application{
			Status: argo.ApplicationStatus{
				Resources: []argo.ResourceStatus{
					{
						Kind:      "ClusterClaim",
						Version:   "v1alpha1",
						Name:      "foo",
						Namespace: "bar",
					},
				},
			},
		}
		provider = GetClusterProvider(app, log)
		Expect(provider).Should(BeNil())
	})

})

func testProvider(
	clusterProvider ClusterProvider,
	cti v1alpha1.ClusterTemplateInstance,
	getResources func(opts ResourceOpts) []runtime.Object,
) {
	It("Returns not ready and err when resource does not exist", func() {
		client := fake.NewFakeClientWithScheme(scheme.Scheme)

		ready, msg, err := clusterProvider.GetClusterStatus(ctx, client, cti)
		Expect(err).Should(HaveOccurred())
		Expect(ready).Should(BeFalse())
		Expect(msg).Should(Equal(""))
	})
	It("Returns not ready when resource condition is false", func() {
		resources := getResources(ResourceOpts{})
		client := fake.NewFakeClientWithScheme(scheme.Scheme, resources...)

		ready, msg, err := clusterProvider.GetClusterStatus(ctx, client, cti)
		Expect(err).NotTo(HaveOccurred())
		Expect(ready).Should(BeFalse())
		Expect(msg).Should(Equal("Not available - foo"))
	})
	It("Returns not ready when resource condition is true but credentials are missing", func() {
		resources := getResources(ResourceOpts{isReady: true})
		client := fake.NewFakeClientWithScheme(scheme.Scheme, resources...)

		ready, msg, err := clusterProvider.GetClusterStatus(ctx, client, cti)
		Expect(err).NotTo(HaveOccurred())
		Expect(ready).Should(BeFalse())
		Expect(msg).Should(Equal("Waiting for pass/kubeconfig secrets"))
	})
	It("Returns not ready when resource condition is true but kubeconfig is missing", func() {
		resources := getResources(ResourceOpts{isReady: true, kubeadmin: true})
		client := fake.NewFakeClientWithScheme(scheme.Scheme, resources...)

		ready, msg, err := clusterProvider.GetClusterStatus(ctx, client, cti)
		Expect(err).NotTo(HaveOccurred())
		Expect(ready).Should(BeFalse())
		Expect(msg).Should(Equal("Waiting for pass/kubeconfig secrets"))
	})
	It("Returns not ready when resource condition is true but kubeadmin is missing", func() {
		resources := getResources(ResourceOpts{isReady: true, kubeconfig: true})
		client := fake.NewFakeClientWithScheme(scheme.Scheme, resources...)

		ready, msg, err := clusterProvider.GetClusterStatus(ctx, client, cti)
		Expect(err).NotTo(HaveOccurred())
		Expect(ready).Should(BeFalse())
		Expect(msg).Should(Equal("Waiting for pass/kubeconfig secrets"))
	})
	It(
		"Returns not ready when resource condition is true and kubeadmin has incorrect format",
		func() {
			resources := getResources(ResourceOpts{
				isReady:          true,
				kubeadmin:        true,
				kubeadminInvalid: true,
				kubeconfig:       true,
			})
			client := fake.NewFakeClientWithScheme(scheme.Scheme, resources...)

			ready, msg, err := clusterProvider.GetClusterStatus(ctx, client, cti)
			Expect(err).To(HaveOccurred())
			Expect(ready).Should(BeFalse())
			Expect(msg).Should(Equal(""))
		},
	)
	It(
		"Returns not ready when resource condition is true and kubeconfig has incorrect format",
		func() {
			resources := getResources(ResourceOpts{
				isReady:           true,
				kubeadmin:         true,
				kubeconfig:        true,
				kubeconfigInvalid: true,
			})
			client := fake.NewFakeClientWithScheme(scheme.Scheme, resources...)

			ready, msg, err := clusterProvider.GetClusterStatus(ctx, client, cti)
			Expect(err).To(HaveOccurred())
			Expect(ready).Should(BeFalse())
			Expect(msg).Should(Equal(""))
		},
	)
	It("Returns ready when resource condition is true and secrets are ready", func() {
		resources := getResources(ResourceOpts{
			isReady:    true,
			kubeadmin:  true,
			kubeconfig: true,
		})
		client := fake.NewFakeClientWithScheme(scheme.Scheme, resources...)

		ready, msg, err := clusterProvider.GetClusterStatus(ctx, client, cti)
		Expect(err).ToNot(HaveOccurred())
		Expect(ready).Should(BeTrue())
		Expect(msg).Should(Equal("Available"))

		kubeadminSecret := &corev1.Secret{}
		err = client.Get(
			ctx,
			kubeClient.ObjectKey{Name: cti.GetKubeadminPassRef(), Namespace: cti.Namespace},
			kubeadminSecret,
		)
		Expect(err).ToNot(HaveOccurred())

		val, ok := kubeadminSecret.Data["password"]
		Expect(ok).Should(BeTrue())
		pass := ""
		err = json.Unmarshal(val, &pass)
		Expect(err).ToNot(HaveOccurred())
		Expect(pass).To(Equal("foo"))

		kubeconfigSecret := &corev1.Secret{}
		err = client.Get(
			ctx,
			kubeClient.ObjectKey{Name: cti.GetKubeconfigRef(), Namespace: cti.Namespace},
			kubeconfigSecret,
		)
		Expect(err).ToNot(HaveOccurred())

		val, ok = kubeconfigSecret.Data["kubeconfig"]
		Expect(ok).Should(BeTrue())
		kubeconfigVal := api.Config{}
		err = yaml.Unmarshal(val, &kubeconfigVal)
		Expect(err).ToNot(HaveOccurred())

	})
}

type ResourceOpts struct {
	isReady           bool
	kubeadmin         bool
	kubeadminInvalid  bool
	kubeconfig        bool
	kubeconfigInvalid bool
}

func getHostedCluster(opts ResourceOpts) []runtime.Object {
	resources := []runtime.Object{}

	conditionStatus := metav1.ConditionFalse
	if opts.isReady {
		conditionStatus = metav1.ConditionTrue
	}
	hostedCluster := &hypershiftv1alpha1.HostedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Status: hypershiftv1alpha1.HostedClusterStatus{
			Conditions: []metav1.Condition{
				{
					Type:               string(hypershiftv1alpha1.HostedClusterAvailable),
					Status:             conditionStatus,
					Reason:             "foo",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	if opts.kubeadmin {
		kubeadminSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubeadminSecret",
				Namespace: "bar",
			},
		}

		hostedCluster.Status.KubeadminPassword = &corev1.LocalObjectReference{
			Name: "kubeadminSecret",
		}

		if !opts.kubeadminInvalid {
			kubeAdminBytes, err := json.Marshal("foo")
			if err != nil {
				Fail(err.Error())
			}
			kubeadminSecret.Data = map[string][]byte{
				"password": kubeAdminBytes,
			}
		}

		resources = append(resources, kubeadminSecret)
	}

	if opts.kubeconfig {
		kubeconfigSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubeconfigSecret",
				Namespace: "bar",
			},
		}

		hostedCluster.Status.KubeConfig = &corev1.LocalObjectReference{
			Name: "kubeconfigSecret",
		}

		if !opts.kubeconfigInvalid {
			kubeconfigFile, err := os.ReadFile("../testutils/kubeconfig_mock.yaml")
			if err != nil {
				Fail(err.Error())
			}
			kubeconfigSecret.Data = map[string][]byte{
				"kubeconfig": kubeconfigFile,
			}
		}

		resources = append(resources, kubeconfigSecret)
	}

	resources = append(resources, hostedCluster)

	return resources
}

func getHostedClusterWithNodePools(opts ResourceOpts) []runtime.Object {
	resources := getHostedCluster(opts)
	conditionStatus := corev1.ConditionFalse
	if opts.isReady {
		conditionStatus = corev1.ConditionTrue
	}
	np := hypershiftv1alpha1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "np1",
			Namespace: "bar",
		},
		Spec: hypershiftv1alpha1.NodePoolSpec{
			ClusterName: "foo",
		},
		Status: hypershiftv1alpha1.NodePoolStatus{
			Conditions: []hypershiftv1alpha1.NodePoolCondition{
				{
					Type:               hypershiftv1alpha1.NodePoolReadyConditionType,
					Status:             conditionStatus,
					Reason:             "foo",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}
	return append(resources, &np)
}

func getClusterClaim(opts ResourceOpts) []runtime.Object {
	resources := []runtime.Object{}

	conditionStatus := corev1.ConditionFalse
	if opts.isReady {
		conditionStatus = corev1.ConditionTrue
	}
	clusterClaim := &hivev1.ClusterClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Status: hivev1.ClusterClaimStatus{
			Conditions: []hivev1.ClusterClaimCondition{
				{
					Type:               hivev1.ClusterClaimPendingCondition,
					Status:             conditionStatus,
					Reason:             "foo",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	clusterDeployment := &hivev1.ClusterDeployment{}

	if opts.isReady {
		clusterClaim.Spec.Namespace = "clusterdeployment"

		clusterDeployment = &hivev1.ClusterDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "clusterdeployment",
				Namespace: "clusterdeployment",
			},
		}

		resources = append(resources, clusterDeployment)
	}

	if opts.kubeadmin {
		kubeadminSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubeadminSecret",
				Namespace: "clusterdeployment",
			},
		}

		clusterDeployment.Spec = hivev1.ClusterDeploymentSpec{
			ClusterMetadata: &hivev1.ClusterMetadata{
				AdminPasswordSecretRef: &corev1.LocalObjectReference{
					Name: "kubeadminSecret",
				},
			},
		}

		if !opts.kubeadminInvalid {
			kubeAdminPassBytes, err := json.Marshal("foo")
			if err != nil {
				Fail(err.Error())
			}
			kubeAdminUserBytes, err := json.Marshal("kubeadmin")
			if err != nil {
				Fail(err.Error())
			}
			kubeadminSecret.Data = map[string][]byte{
				"password": kubeAdminPassBytes,
				"username": kubeAdminUserBytes,
			}
		}

		resources = append(resources, kubeadminSecret)
	}

	if opts.kubeconfig {
		kubeconfigSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubeconfigSecret",
				Namespace: "clusterdeployment",
			},
		}

		if clusterDeployment.Spec.ClusterMetadata == nil {
			clusterDeployment.Spec.ClusterMetadata = &hivev1.ClusterMetadata{}
		}

		clusterDeployment.Spec.ClusterMetadata.AdminKubeconfigSecretRef = corev1.LocalObjectReference{
			Name: "kubeconfigSecret",
		}

		if !opts.kubeconfigInvalid {
			kubeconfigFile, err := os.ReadFile("../testutils/kubeconfig_mock.yaml")
			if err != nil {
				Fail(err.Error())
			}
			kubeconfigSecret.Data = map[string][]byte{
				"kubeconfig": kubeconfigFile,
			}
		}

		resources = append(resources, kubeconfigSecret)
	}

	resources = append(resources, clusterClaim)

	return resources
}

func getClusterDeployment(opts ResourceOpts) []runtime.Object {
	resources := []runtime.Object{}

	conditionStatus := corev1.ConditionFalse
	if opts.isReady {
		conditionStatus = corev1.ConditionTrue
	}
	clusterDeployment := &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Status: hivev1.ClusterDeploymentStatus{
			Conditions: []hivev1.ClusterDeploymentCondition{
				{
					Type:               hivev1.ClusterInstallCompletedClusterDeploymentCondition,
					Status:             conditionStatus,
					Reason:             "foo",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	if opts.kubeadmin {
		kubeadminSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubeadminSecret",
				Namespace: "bar",
			},
		}

		clusterDeployment.Spec = hivev1.ClusterDeploymentSpec{
			ClusterMetadata: &hivev1.ClusterMetadata{
				AdminPasswordSecretRef: &corev1.LocalObjectReference{
					Name: "kubeadminSecret",
				},
			},
		}

		if !opts.kubeadminInvalid {
			kubeAdminPassBytes, err := json.Marshal("foo")
			if err != nil {
				Fail(err.Error())
			}
			kubeAdminUserBytes, err := json.Marshal("kubeadmin")
			if err != nil {
				Fail(err.Error())
			}
			kubeadminSecret.Data = map[string][]byte{
				"password": kubeAdminPassBytes,
				"username": kubeAdminUserBytes,
			}
		}

		resources = append(resources, kubeadminSecret)
	}

	if opts.kubeconfig {
		kubeconfigSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubeconfigSecret",
				Namespace: "bar",
			},
		}

		if clusterDeployment.Spec.ClusterMetadata == nil {
			clusterDeployment.Spec.ClusterMetadata = &hivev1.ClusterMetadata{}
		}

		clusterDeployment.Spec.ClusterMetadata.AdminKubeconfigSecretRef = corev1.LocalObjectReference{
			Name: "kubeconfigSecret",
		}

		if !opts.kubeconfigInvalid {
			kubeconfigFile, err := os.ReadFile("../testutils/kubeconfig_mock.yaml")
			if err != nil {
				Fail(err.Error())
			}
			kubeconfigSecret.Data = map[string][]byte{
				"kubeconfig": kubeconfigFile,
			}
		}

		resources = append(resources, kubeconfigSecret)
	}

	resources = append(resources, clusterDeployment)

	return resources
}
