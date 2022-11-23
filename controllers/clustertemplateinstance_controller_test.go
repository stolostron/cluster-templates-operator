package controllers

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/testutils"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 10
	duration = time.Second * 10
	interval = time.Millisecond * 250
)

func deleteResource(obj client.Object) {
	Expect(k8sClient.Delete(ctx, obj)).Should(Succeed())
	Eventually(func() bool {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
		return apierrors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue())
}

var _ = Describe("ClusterTemplateInstance controller", func() {
	Context("Initial ClusterTemplateInstance Status", func() {
		ct := &v1alpha1.ClusterTemplate{}
		cti := &v1alpha1.ClusterTemplateInstance{}
		ctiLookupKey := types.NamespacedName{}

		BeforeEach(func() {
			ct = testutils.GetCT(false, false)
			Expect(k8sClient.Create(ctx, ct)).Should(Succeed())
			cti, ctiLookupKey = testutils.GetCTI()
			Expect(k8sClient.Create(ctx, cti)).Should(Succeed())
		})

		AfterEach(func() {
			deleteResource(cti)
			deleteResource(ct)
		})

		It("Should create default conditions", func() {
			Eventually(func() int {
				err := k8sClient.Get(ctx, ctiLookupKey, cti)
				if err != nil {
					return 0
				}
				return len(cti.Status.Conditions)
			}, timeout, interval).Should(Equal(5))
		})
	})

	Context("Missing CT", func() {
		cti := &v1alpha1.ClusterTemplateInstance{}
		ctiLookupKey := types.NamespacedName{}

		BeforeEach(func() {
			cti, ctiLookupKey = testutils.GetCTI()
			Expect(k8sClient.Create(ctx, cti)).Should(Succeed())
		})

		AfterEach(func() {
			deleteResource(cti)
		})
		It("Should have failed phase when ct does not exist", func() {
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ctiLookupKey, cti)
				if err != nil {
					return false
				}
				return cti.Status.Phase == v1alpha1.FailedPhase
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Cluster definition phase", func() {
		cti := &v1alpha1.ClusterTemplateInstance{}
		ctiLookupKey := types.NamespacedName{}

		ct := &v1alpha1.ClusterTemplate{}
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "argocd",
			},
		}

		BeforeEach(func() {
			ct = testutils.GetCT(false, false)
			Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

			cti, ctiLookupKey = testutils.GetCTI()

			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			deleteResource(cti)
			deleteResource(ct)
		})

		It("Should create cluster definition argo app", func() {
			Expect(k8sClient.Create(ctx, cti)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ctiLookupKey, cti)
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
			app, err := cti.GetDay1Application(ctx, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(app).ShouldNot(BeNil())
		})
	})

	/*
		Context("Cluster status phase", func() {
			cti := &v1alpha1.ClusterTemplateInstance{}
			ctiLookupKey := types.NamespacedName{}
			ct := &v1alpha1.ClusterTemplate{}

			BeforeEach(func() {

				ct = testutils.GetCT(true, false)
				Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

				cti, ctiLookupKey = testutils.GetCTI()
				Expect(k8sClient.Create(ctx, cti)).Should(Succeed())

				app, err := cti.GetDay1Application(ctx, k8sClient)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(app).ShouldNot(BeNil())

				app.Status.Health = argo.HealthStatus{
					Status: health.HealthStatusDegraded,
				}

				Expect(k8sClient.Update(ctx, app)).Should(Succeed())
			})

			AfterEach(func() {
				Expect(k8sClient.Delete(ctx, cti)).Should(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, ctiLookupKey, cti)
					return apierrors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, ct)).Should(Succeed())
			})

			It("Detects app degraded", func() {
				Eventually(func() bool {
					err := k8sClient.Get(ctx, ctiLookupKey, cti)
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

			It("Handles unknwn provider", func() {
				app, err := cti.GetDay1Application(ctx, k8sClient)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(app).ShouldNot(BeNil())

				app.Status.Health = argo.HealthStatus{
					Status: health.HealthStatusHealthy,
				}

				Expect(k8sClient.Update(ctx, app)).Should(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, ctiLookupKey, cti)
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

			It("Handles known provider", func() {
				app, err := cti.GetDay1Application(ctx, k8sClient)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(app).ShouldNot(BeNil())

				app.Status.Health = argo.HealthStatus{
					Status: health.HealthStatusHealthy,
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
					err := k8sClient.Get(ctx, ctiLookupKey, cti)
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
		})

		Context("Cluster setup chart phase", func() {
			cti := &v1alpha1.ClusterTemplateInstance{}
			cti, _ = testutils.GetCTI()

			It("Creates dynamic roles", func() {
				// So far this is partial mocked test only.
				// Once we run full-flow test including cluster creaton, this should be
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
					types.NamespacedName{Name: cti.Name + "-rolebinding-managed", Namespace: cti.Namespace},
					roleBinding,
				)).Should(Succeed())
				Expect(roleBinding.RoleRef.Kind).Should(Equal("Role"))
				Expect(roleBinding.RoleRef.Name).Should(Equal(role.Name))
				Expect(len(roleBinding.Subjects)).Should(Equal(3))
			})
	*/

	/*
		ct := &v1alpha1.ClusterTemplate{}
		helmRepoCR := &openshiftAPI.HelmChartRepository{}
		helRepoServer := &httptest.Server{}

		task := &tekton.ClusterTask{}
		pipeline := &tekton.Pipeline{}
		pipelineRun := &tekton.PipelineRun{}

		kubeConfigSecret := &corev1.Secret{}
		kubeAdminSecret := &corev1.Secret{}
		ns := &corev1.Namespace{}

		BeforeEach(func() {
			helRepoServer = helmserver.StartHelmRepoServer()
			helmRepoCR = testutils.GetHelmRepo(helRepoServer.URL)
			Expect(k8sClient.Create(ctx, helmRepoCR)).Should(Succeed())

			cTemp, err := testutils.GetCT(true, true)
			if err != nil {
				Fail(err.Error())
			}
			ct = cTemp
			Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

			task = testutils.GetClusterTask()
			Expect(k8sClient.Create(ctx, task)).Should(Succeed())

			pipeline, ns = testutils.GetPipeline()
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
			Expect(k8sClient.Create(ctx, pipeline)).Should(Succeed())

			cti, ctiLookupKey = testutils.GetCTI()
			Expect(k8sClient.Create(ctx, cti)).Should(Succeed())

			kubeConfigSecret, err = testutils.GetKubeconfigSecret()
			if err != nil {
				Fail(err.Error())
			}
			Expect(k8sClient.Create(ctx, kubeConfigSecret)).Should(Succeed())

			kubeAdminSecret, err = testutils.GetKubeadminSecret()
			if err != nil {
				Fail(err.Error())
			}
			Expect(k8sClient.Create(ctx, kubeAdminSecret)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, cti)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ctiLookupKey, cti)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
			Expect(k8sClient.Delete(ctx, helmRepoCR)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, ct)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, kubeConfigSecret)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, kubeAdminSecret)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, pipeline)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, task)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
			helRepoServer.Close()
		})

		It("Detects when cluster setup finished", func() {
			hostedCluster := &hypershiftv1alpha1.HostedCluster{}
			Eventually(func() error {
				return k8sClient.Get(ctx, ctiLookupKey, hostedCluster)
			}, timeout, interval).Should(Succeed())
			testutils.SetHostedClusterReady(hostedCluster, kubeConfigSecret.Name, kubeAdminSecret.Name)
			Expect(k8sClient.Status().Update(ctx, hostedCluster)).Should(Succeed())

			Eventually(func() bool {
				pipelineRuns := &tekton.PipelineRunList{}
				err := k8sClient.List(
					ctx,
					pipelineRuns,
					&client.ListOptions{
						Namespace: cti.Namespace,
					},
				)
				if err != nil {
					return false
				}
				for _, pRun := range pipelineRuns.Items {
					if pRun.Labels[clustersetup.ClusterSetupInstanceLabel] == cti.Name {
						pipelineRun = &pRun
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			testutils.SetPipelineRunSucceeded(pipelineRun)
			Expect(k8sClient.Status().Update(ctx, pipelineRun)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, ctiLookupKey, cti)
				if err != nil {
					return false
				}
				clusterCondition := meta.FindStatusCondition(
					cti.Status.Conditions,
					string(v1alpha1.SetupPipelineSucceeded),
				)
				if clusterCondition == nil {
					return false
				}
				return clusterCondition.Status == metav1.ConditionTrue
			}, timeout, interval).Should(BeTrue())
			Expect(cti.Status.Phase).Should(Equal(v1alpha1.ReadyPhase))
			Expect(cti.Status.AdminPassword.Name).Should(Equal(cti.Name + "-admin-password"))
			Expect(cti.Status.Kubeconfig.Name).Should(Equal(cti.Name + "-admin-kubeconfig"))

		})
	*/

})
