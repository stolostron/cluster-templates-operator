package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/argoproj/gitops-engine/pkg/health"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/testutils"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	hypershift "github.com/openshift/hypershift/api/v1alpha1"

	"github.com/kubernetes-client/go-base/config/api"
	"gopkg.in/yaml.v3"
)

var _ = Describe("ClusterTemplateInstance controller", func() {
	Context("Initial ClusterTemplateInstance Status", func() {
		ct := &v1alpha1.ClusterTemplate{}
		cti := &v1alpha1.ClusterTemplateInstance{}

		BeforeEach(func() {
			ct = testutils.GetCT(false)
			Expect(k8sClient.Create(ctx, ct)).Should(Succeed())
			cti = testutils.GetCTI()
			Expect(k8sClient.Create(ctx, cti)).Should(Succeed())
		})

		AfterEach(func() {
			testutils.DeleteResource(ctx, cti, k8sClient)
			testutils.DeleteResource(ctx, ct, k8sClient)
		})

		It("Should create default conditions", func() {
			Eventually(func() int {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cti), cti)
				if err != nil {
					return 0
				}
				return len(cti.Status.Conditions)
			}, timeout, interval).Should(Equal(5))
		})
	})

	Context("Missing CT", func() {
		cti := &v1alpha1.ClusterTemplateInstance{}

		BeforeEach(func() {
			cti = testutils.GetCTI()
			Expect(k8sClient.Create(ctx, cti)).Should(Succeed())
		})

		AfterEach(func() {
			testutils.DeleteResource(ctx, cti, k8sClient)
		})
		It("Should have failed phase when ct does not exist", func() {
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cti), cti)
				if err != nil {
					return false
				}
				return cti.Status.Phase == v1alpha1.FailedPhase
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Cluster definition phase", func() {
		cti := &v1alpha1.ClusterTemplateInstance{}

		ct := &v1alpha1.ClusterTemplate{}
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "argocd",
			},
		}

		BeforeEach(func() {
			ct = testutils.GetCT(false)
			Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

			cti = testutils.GetCTI()

			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			testutils.DeleteResource(ctx, cti, k8sClient)
			testutils.DeleteResource(ctx, ct, k8sClient)
		})

		It("Should create cluster definition argo app", func() {
			Expect(k8sClient.Create(ctx, cti)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cti), cti)
				if err != nil {
					return false
				}
				clusterDefinitionCondition := meta.FindStatusCondition(
					cti.Status.Conditions,
					string(v1alpha1.ClusterDefinitionCreated),
				)
				if clusterDefinitionCondition == nil {
					return false
				}
				return clusterDefinitionCondition.Status == metav1.ConditionTrue
			}, timeout, interval).Should(BeTrue())
			Expect(cti.Status.Phase).Should(Equal(v1alpha1.ClusterInstallingPhase))
			app, err := cti.GetDay1Application(ctx, k8sClient, "argocd")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(app).ShouldNot(BeNil())
		})
	})

	Context("Cluster status phase", func() {
		cti := &v1alpha1.ClusterTemplateInstance{}
		ct := &v1alpha1.ClusterTemplate{}
		app := &argo.Application{}
		resourcesToDelete := []client.Object{}

		BeforeEach(func() {

			ct = testutils.GetCT(false)
			Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

			clusterti := testutils.GetCTI()
			Expect(k8sClient.Create(ctx, clusterti)).Should(Succeed())

			Eventually(func() bool {
				k8sClient.Get(ctx, client.ObjectKeyFromObject(clusterti), cti)
				return cti.Status.ClusterTemplateSpec != nil
			}, timeout, interval).Should(BeTrue())
			var err error
			app, err = cti.GetDay1Application(ctx, k8sClient, "argocd")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(app).ShouldNot(BeNil())
			resourcesToDelete = []client.Object{}
		})

		AfterEach(func() {
			testutils.DeleteResource(ctx, cti, k8sClient)
			testutils.DeleteResource(ctx, ct, k8sClient)
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app)).Should(Succeed())
			app.Finalizers = []string{}
			Expect(k8sClient.Update(ctx, app)).Should(Succeed())
			testutils.EnsureResourceDoesNotExist(ctx, app, k8sClient)
			for _, obj := range resourcesToDelete {
				testutils.DeleteResource(ctx, obj, k8sClient)
			}
		})

		It("Detects app degraded", func() {
			app.Status.Health = argo.HealthStatus{
				Status: health.HealthStatusDegraded,
			}

			Expect(k8sClient.Update(ctx, app)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cti), cti)
				if err != nil {
					return false
				}
				clusterCondition := meta.FindStatusCondition(
					cti.Status.Conditions,
					string(v1alpha1.ClusterInstallSucceeded),
				)
				if clusterCondition == nil {
					return false
				}
				return clusterCondition.Status == metav1.ConditionFalse &&
					clusterCondition.Reason == string(v1alpha1.ApplicationDegraded)
			}, timeout, interval).Should(BeTrue())
		})

		It("Detects app error", func() {
			app.Status.Health = argo.HealthStatus{
				Status: health.HealthStatusHealthy,
			}
			currentTime := metav1.Now()
			app.Status.Conditions = []argo.ApplicationCondition{
				{
					Type:               argo.ApplicationConditionSyncError,
					Message:            "foo msg",
					LastTransitionTime: &currentTime,
				},
			}
			app.Status.OperationState = &argo.OperationState{
				StartedAt: currentTime,
				Phase:     synccommon.OperationSucceeded,
			}

			Expect(k8sClient.Update(ctx, app)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cti), cti)
				if err != nil {
					return false
				}
				clusterCondition := meta.FindStatusCondition(
					cti.Status.Conditions,
					string(v1alpha1.ClusterInstallSucceeded),
				)
				if clusterCondition == nil {
					return false
				}
				return clusterCondition.Status == metav1.ConditionFalse &&
					clusterCondition.Reason == string(v1alpha1.ApplicationError) &&
					clusterCondition.Message == "foo msg"
			}, timeout, interval).Should(BeTrue())
		})

		It("Handles unknown provider", func() {
			app.Status.Health = argo.HealthStatus{
				Status: health.HealthStatusHealthy,
			}
			app.Status.OperationState = &argo.OperationState{
				StartedAt: metav1.Now(),
				Phase:     synccommon.OperationSucceeded,
			}

			Expect(k8sClient.Update(ctx, app)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cti), cti)
				if err != nil {
					return false
				}
				clusterCondition := meta.FindStatusCondition(
					cti.Status.Conditions,
					string(v1alpha1.ClusterInstallSucceeded),
				)
				if clusterCondition == nil {
					return false
				}

				return clusterCondition.Status == metav1.ConditionFalse &&
					clusterCondition.Reason == string(v1alpha1.ClusterProviderDetectionFailed)
			}, timeout, interval).Should(BeTrue())
		})

		It("Handles known provider - but resource is missing", func() {
			app.Status.Health = argo.HealthStatus{
				Status: health.HealthStatusHealthy,
			}
			app.Status.OperationState = &argo.OperationState{
				StartedAt: metav1.Now(),
				Phase:     synccommon.OperationSucceeded,
			}
			app.Status.Resources = []argo.ResourceStatus{
				{
					Group:     "hypershift.openshift.io",
					Version:   "v1alpha1",
					Kind:      "HostedCluster",
					Name:      "foo",
					Namespace: "bar",
				},
			}

			Expect(k8sClient.Update(ctx, app)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cti), cti)
				if err != nil {
					return false
				}
				clusterCondition := meta.FindStatusCondition(
					cti.Status.Conditions,
					string(v1alpha1.ClusterInstallSucceeded),
				)
				if clusterCondition == nil {
					return false
				}
				return clusterCondition.Status == metav1.ConditionFalse &&
					clusterCondition.Reason == string(v1alpha1.ClusterStatusFailed)
			}, timeout, interval).Should(BeTrue())
		})

		It("Handles known provider - hc not ready - missing kubeconfig/pass", func() {
			hc := &hypershift.HostedCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HostedCluster",
					APIVersion: "hypershift.openshift.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: hypershift.HostedClusterSpec{
					Release: hypershift.Release{
						Image: "foo",
					},
					Platform: hypershift.PlatformSpec{
						Type: hypershift.AWSPlatform,
					},
					Networking: hypershift.ClusterNetworking{
						NetworkType: hypershift.OpenShiftSDN,
					},
					Etcd: hypershift.EtcdSpec{
						ManagementType: hypershift.Managed,
					},
					Services: []hypershift.ServicePublishingStrategyMapping{},
					PullSecret: corev1.LocalObjectReference{
						Name: "pullsecret",
					},
					SSHKey: corev1.LocalObjectReference{
						Name: "sshsecret",
					},
				},
			}
			resourcesToDelete = append(resourcesToDelete, hc)

			Expect(k8sClient.Create(ctx, hc)).Should(Succeed())

			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hc), hc)).Should(Succeed())

			hc.Status.Conditions = []metav1.Condition{
				{
					Type:               string(hypershift.HostedClusterAvailable),
					Status:             metav1.ConditionTrue,
					Reason:             "foo",
					Message:            "foo",
					LastTransitionTime: metav1.Now(),
				},
			}
			Expect(k8sClient.Status().Update(ctx, hc)).Should(Succeed())

			app.Status = argo.ApplicationStatus{
				Health: argo.HealthStatus{
					Status: health.HealthStatusHealthy,
				},
				OperationState: &argo.OperationState{
					StartedAt: metav1.Now(),
					Phase:     synccommon.OperationSucceeded,
					Operation: argo.Operation{},
				},
				Resources: []argo.ResourceStatus{
					{
						Group:     "hypershift.openshift.io",
						Version:   "v1alpha1",
						Kind:      "HostedCluster",
						Name:      "foo",
						Namespace: "default",
					},
				},
			}

			Expect(k8sClient.Update(ctx, app)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cti), cti)
				if err != nil {
					return false
				}
				clusterCondition := meta.FindStatusCondition(
					cti.Status.Conditions,
					string(v1alpha1.ClusterInstallSucceeded),
				)
				if clusterCondition == nil {
					return false
				}
				return clusterCondition.Status == metav1.ConditionFalse &&
					clusterCondition.Reason == string(v1alpha1.ClusterInstalling)
			}, timeout, interval).Should(BeTrue())
		})

		It("Handles known provider - hc is ready", func() {
			hc := &hypershift.HostedCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HostedCluster",
					APIVersion: "hypershift.openshift.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: hypershift.HostedClusterSpec{
					Release: hypershift.Release{
						Image: "foo",
					},
					Platform: hypershift.PlatformSpec{
						Type: hypershift.AWSPlatform,
					},
					Networking: hypershift.ClusterNetworking{
						NetworkType: hypershift.OpenShiftSDN,
					},
					Etcd: hypershift.EtcdSpec{
						ManagementType: hypershift.Managed,
					},
					Services: []hypershift.ServicePublishingStrategyMapping{},
					PullSecret: corev1.LocalObjectReference{
						Name: "pullsecret",
					},
					SSHKey: corev1.LocalObjectReference{
						Name: "sshsecret",
					},
				},
			}
			resourcesToDelete = append(resourcesToDelete, hc)
			Expect(k8sClient.Create(ctx, hc)).Should(Succeed())

			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hc), hc)).Should(Succeed())

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
			kubeconfigSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubeconfig",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"kubeconfig": data,
				},
			}
			Expect(k8sClient.Create(ctx, kubeconfigSecret)).Should(Succeed())
			resourcesToDelete = append(resourcesToDelete, kubeconfigSecret)

			adminpassSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "adminpass",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"password": []byte("pass"),
				},
			}
			Expect(k8sClient.Create(ctx, adminpassSecret)).Should(Succeed())
			resourcesToDelete = append(resourcesToDelete, adminpassSecret)

			hc.Status.KubeConfig = &corev1.LocalObjectReference{
				Name: kubeconfigSecret.Name,
			}
			hc.Status.KubeadminPassword = &corev1.LocalObjectReference{
				Name: adminpassSecret.Name,
			}

			hc.Status.Conditions = []metav1.Condition{
				{
					Type:               string(hypershift.HostedClusterAvailable),
					Status:             metav1.ConditionTrue,
					Reason:             "foo",
					Message:            "foo",
					LastTransitionTime: metav1.Now(),
				},
			}
			Expect(k8sClient.Status().Update(ctx, hc)).Should(Succeed())

			app.Status = argo.ApplicationStatus{
				Health: argo.HealthStatus{
					Status: health.HealthStatusHealthy,
				},
				OperationState: &argo.OperationState{
					StartedAt: metav1.Now(),
					Phase:     synccommon.OperationSucceeded,
					Operation: argo.Operation{},
				},
				Resources: []argo.ResourceStatus{
					{
						Group:     "hypershift.openshift.io",
						Version:   "v1alpha1",
						Kind:      "HostedCluster",
						Name:      "foo",
						Namespace: "default",
					},
				},
			}

			Expect(k8sClient.Update(ctx, app)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cti), cti)
				if err != nil {
					return false
				}
				clusterCondition := meta.FindStatusCondition(
					cti.Status.Conditions,
					string(v1alpha1.ClusterInstallSucceeded),
				)
				if clusterCondition == nil {
					return false
				}
				return clusterCondition.Status == metav1.ConditionTrue &&
					clusterCondition.Reason == string(v1alpha1.ClusterInstalled)
			}, timeout, interval).Should(BeTrue())

		})
	})

	Context("Cluster setup create phase", func() {
		ct := &v1alpha1.ClusterTemplate{}
		cti := &v1alpha1.ClusterTemplateInstance{}
		BeforeEach(func() {
			ct = testutils.GetCT(true)
			cti = testutils.GetCTI()
		})

		It("Creates day2 apps", func() {
			cti.Status = v1alpha1.ClusterTemplateInstanceStatus{
				Conditions: []metav1.Condition{
					{
						Type:   string(v1alpha1.ArgoClusterAdded),
						Status: metav1.ConditionTrue,
					},
					{
						Type:   string(v1alpha1.ClusterSetupCreated),
						Status: metav1.ConditionFalse,
					},
				},
				ClusterTemplateSpec: &ct.Spec,
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
			kubeconfigSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cti.GetKubeconfigRef(),
					Namespace: cti.Namespace,
				},
				Data: map[string][]byte{
					"kubeconfig": data,
				},
			}

			client := fake.NewFakeClientWithScheme(scheme.Scheme, kubeconfigSecret)
			reconciler := &ClusterTemplateInstanceReconciler{
				Client: client,
			}
			err = reconciler.reconcileClusterSetupCreate(ctx, cti)
			Expect(err).Should(BeNil())
			apps, err := cti.GetDay2Applications(ctx, client, "argocd")
			Expect(err).Should(BeNil())
			Expect(len(apps.Items)).Should(Equal(1))
			Expect(apps.Items[0].Spec.Destination.Server).Should(Equal("foo-server"))

			clusterSetupCreatedCondition := meta.FindStatusCondition(
				cti.Status.Conditions,
				string(v1alpha1.ClusterSetupCreated),
			)
			Expect(clusterSetupCreatedCondition.Status).Should(Equal(metav1.ConditionTrue))
		})
	})

	Context("Cluster setup create phase", func() {
		ct := &v1alpha1.ClusterTemplate{}
		cti := &v1alpha1.ClusterTemplateInstance{}
		BeforeEach(func() {
			ct = testutils.GetCT(true)
			cti = testutils.GetCTI()
		})
		It("Detects running day2 app", func() {
			cti.Status = v1alpha1.ClusterTemplateInstanceStatus{
				Conditions: []metav1.Condition{
					{
						Type:   string(v1alpha1.ClusterSetupCreated),
						Status: metav1.ConditionTrue,
					},
					{
						Type:   string(v1alpha1.ClusterSetupSucceeded),
						Status: metav1.ConditionFalse,
					},
				},
				ClusterTemplateSpec: &ct.Spec,
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
			kubeconfigSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cti.GetKubeconfigRef(),
					Namespace: cti.Namespace,
				},
				Data: map[string][]byte{
					"kubeconfig": data,
				},
			}

			client := fake.NewFakeClientWithScheme(scheme.Scheme, kubeconfigSecret)
			err = cti.CreateDay2Applications(ctx, client, "argocd")
			Expect(err).Should(BeNil())
			reconciler := &ClusterTemplateInstanceReconciler{
				Client: client,
			}
			err = reconciler.reconcileClusterSetup(ctx, cti)
			Expect(err).Should(BeNil())
			clusterSetupSucceededCondition := meta.FindStatusCondition(
				cti.Status.Conditions,
				string(v1alpha1.ClusterSetupSucceeded),
			)
			Expect(clusterSetupSucceededCondition.Status).Should(Equal(metav1.ConditionFalse))
			Expect(
				clusterSetupSucceededCondition.Reason,
			).Should(Equal(string(v1alpha1.ClusterSetupRunning)))
		})
	})

	Context("Credentials phase", func() {
		cti := &v1alpha1.ClusterTemplateInstance{}
		cti = testutils.GetCTI()

		It("Creates dynamic roles", func() {
			// So far this is partial mocked test only.
			// Once we run full-flow test including cluster creation, this should be
			// black-box tested the same way as performed at previous phases.

			reconciler := &ClusterTemplateInstanceReconciler{}

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
			}
			client := fake.NewFakeClientWithScheme(scheme.Scheme, objs...)
			reconciler.ReconcileDynamicRoles(ctx, client, cti)

			role := &rbacv1.Role{}
			Expect(client.Get(
				ctx,
				types.NamespacedName{Name: cti.Name + "-role-managed", Namespace: cti.Namespace},
				role,
			)).Should(Succeed())
			Expect(len(role.Rules)).Should(Equal(1))
			Expect(len(role.Rules[0].ResourceNames)).Should(Equal(2))

			roleBinding := &rbacv1.RoleBinding{}
			Expect(client.Get(
				ctx,
				types.NamespacedName{
					Name:      cti.Name + "-rolebinding-managed",
					Namespace: cti.Namespace,
				},
				roleBinding,
			)).Should(Succeed())
			Expect(roleBinding.RoleRef.Kind).Should(Equal("Role"))
			Expect(roleBinding.RoleRef.Name).Should(Equal(role.Name))
			Expect(len(roleBinding.Subjects)).Should(Equal(3))
		})
	})
})
