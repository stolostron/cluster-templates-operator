package controllers

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/gitops-engine/pkg/health"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/testutils"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ClusterTemplateInstance controller", func() {
	const (
		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("Initial ClusterTemplateInstance Status", func() {
		cti := &v1alpha1.ClusterTemplateInstance{}
		ctiLookupKey := types.NamespacedName{}

		BeforeEach(func() {
			cti, ctiLookupKey = testutils.GetCTI()
			Expect(k8sClient.Create(ctx, cti)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, cti)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ctiLookupKey, cti)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
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
				Name: "openshift-gitops",
			},
		}

		BeforeEach(func() {
			ct = testutils.GetCT(false, false)
			Expect(k8sClient.Create(ctx, ct)).Should(Succeed())

			cti, ctiLookupKey = testutils.GetCTI()

			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, cti)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ctiLookupKey, cti)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
			Expect(k8sClient.Delete(ctx, ct)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
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
				return clusterCondition.Status == metav1.ConditionFalse && clusterCondition.Reason == string(v1alpha1.ApplicationDegraded)
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
				return clusterCondition.Status == metav1.ConditionFalse && clusterCondition.Reason == string(v1alpha1.ClusterProviderDetectionFailed)
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
				return clusterCondition.Status == metav1.ConditionFalse && clusterCondition.Reason == string(v1alpha1.ClusterStatusFailed)
			}, timeout, interval).Should(BeTrue())
		})
	})

	/*
		Context("Cluster setup chart phase", func() {
			cti := &v1alpha1.ClusterTemplateInstance{}
			ctiLookupKey := types.NamespacedName{}

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

		})
	*/
})
