/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kubernetes-client/go-base/config/api"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1alpha1 "github.com/rawagner/cluster-templates-operator/api/v1alpha1"

	"github.com/rawagner/cluster-templates-operator/clusterprovider"
	"github.com/rawagner/cluster-templates-operator/clustersetup"
	"github.com/rawagner/cluster-templates-operator/helm"
	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	openshiftAPI "github.com/openshift/api/helm/v1beta1"
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterTemplateInstanceReconciler reconciles a ClusterTemplateInstance object
type ClusterTemplateInstanceReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	HelmClient *helm.HelmClient
}

const clusterTemplateInstanceFinalizer = "clustertemplateinstance.openshift.io/finalizer"

// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplateinstances,verbs=*
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplateinstances/status,verbs=*
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates,verbs=*
// +kubebuilder:rbac:groups=hypershift.openshift.io,resources=*,verbs=*
// +kubebuilder:rbac:groups=hive.openshift.io,resources=*,verbs=*
// +kubebuilder:rbac:groups=helm.openshift.io,resources=helmchartrepositories,verbs=get;list;watch
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelines,verbs=get;list;watch
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=*
// +kubebuilder:rbac:groups="",resources=secrets,verbs=*

func (r *ClusterTemplateInstanceReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	clusterTemplateInstance := &v1alpha1.ClusterTemplateInstance{}
	if err := r.Get(ctx, req.NamespacedName, clusterTemplateInstance); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(
				"clustertemplateinstance not found, aborting reconcile",
				"name",
				req.NamespacedName,
			)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf(
			"failed to get clustertemplateinstance %q: %w",
			req.NamespacedName,
			err,
		)
	}

	if clusterTemplateInstance.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(
			clusterTemplateInstance,
			clusterTemplateInstanceFinalizer,
		) {
			rel, err := r.HelmClient.GetRelease(clusterTemplateInstance.Name)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf(
					"failed to get helm release %q: %w",
					req.NamespacedName,
					err,
				)
			}

			if rel != nil {
				_, err = r.HelmClient.UninstallRelease(clusterTemplateInstance.Name)
				if err != nil {
					return ctrl.Result{}, fmt.Errorf(
						"failed to uninstall helm release %q: %w",
						req.NamespacedName,
						err,
					)
				}
			}

			controllerutil.RemoveFinalizer(
				clusterTemplateInstance,
				clusterTemplateInstanceFinalizer,
			)
			if err := r.Update(ctx, clusterTemplateInstance); err != nil {
				return ctrl.Result{}, fmt.Errorf(
					"failed to remove finalizer on clustertemplateinstance %q: %w",
					req.NamespacedName,
					err,
				)
			}
		}
		log.Info("Deleted clustertemplateinstance", "name", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(
		clusterTemplateInstance,
		clusterTemplateInstanceFinalizer,
	) {
		controllerutil.AddFinalizer(clusterTemplateInstance, clusterTemplateInstanceFinalizer)
		if err := r.Update(ctx, clusterTemplateInstance); err != nil {
			return ctrl.Result{}, err
		}
	}

	if len(clusterTemplateInstance.Status.Conditions) == 0 {
		SetDefaultConditions(clusterTemplateInstance)
	}

	if err := r.reconcile(ctx, clusterTemplateInstance); err != nil {
		return ctrl.Result{}, err
	}

	requeue := true
	if clusterTemplateInstance.Status.CompletionTime != nil {
		requeue = false
		installCondition := meta.FindStatusCondition(
			clusterTemplateInstance.Status.Conditions,
			v1alpha1.InstallSucceeded,
		)
		setupCondition := meta.FindStatusCondition(
			clusterTemplateInstance.Status.Conditions,
			v1alpha1.SetupSucceeded,
		)

		if installCondition.Status == metav1.ConditionTrue &&
			setupCondition.Status == metav1.ConditionTrue {
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               v1alpha1.Ready,
				Status:             metav1.ConditionTrue,
				Reason:             v1alpha1.ClusterReadyReason,
				Message:            "Cluster is ready",
				LastTransitionTime: metav1.Now(),
			})
		} else {
			reason := v1alpha1.ClusterSetupFailedReason
			if installCondition.Status == metav1.ConditionFalse {
				reason = v1alpha1.ClusterInstallFailedReason
			}
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               v1alpha1.Ready,
				Status:             metav1.ConditionFalse,
				Reason:             reason,
				Message:            "Failed",
				LastTransitionTime: metav1.Now(),
			})
		}
	}

	if err := r.Status().Update(ctx, clusterTemplateInstance); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update status of clustertemplateinstance %q: %w",
			req.NamespacedName,
			err,
		)
	}

	return ctrl.Result{Requeue: requeue, RequeueAfter: 60 * time.Second}, nil
}

func (r *ClusterTemplateInstanceReconciler) reconcile(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	clusterTemplate := v1alpha1.ClusterTemplate{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: clusterTemplateInstance.Spec.ClusterTemplateRef}, &clusterTemplate); err != nil {
		return fmt.Errorf("failed to fetch clustertemplate %q", err)
	}

	if err := r.reconcileClusterCreate(ctx, clusterTemplateInstance, clusterTemplate); err != nil {
		return fmt.Errorf("failed to create cluster %q", err)
	}

	if err := r.reconcileClusterStatus(
		ctx,
		clusterTemplateInstance,
	); err != nil {
		return fmt.Errorf("failed to reconcile cluster status %q", err)
	}

	if err := r.reconcileClusterSetup(ctx, clusterTemplateInstance, clusterTemplate); err != nil {
		return fmt.Errorf("failed to reconcile cluster setup %q", err)
	}

	if err := r.reconcileClusterCredentials(ctx, clusterTemplateInstance); err != nil {
		return fmt.Errorf("failed to reconcile cluster credentials %q", err)
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterCreate(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	clusterTemplate v1alpha1.ClusterTemplate,
) error {
	log := ctrl.LoggerFrom(ctx)
	condition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		v1alpha1.InstallSucceeded,
	)

	if condition.Status == metav1.ConditionFalse &&
		condition.Reason != v1alpha1.HelmReleaseInstallingReason {
		log.Info(
			"Create cluster from clustertemplateinstance",
			"name",
			clusterTemplateInstance.Name,
		)

		helmRepositories := &openshiftAPI.HelmChartRepositoryList{}
		if err := r.Client.List(ctx, helmRepositories, &client.ListOptions{}); err != nil {
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               v1alpha1.InstallSucceeded,
				Status:             metav1.ConditionFalse,
				Reason:             v1alpha1.HelmChartRepoErrReason,
				Message:            "Failed to list helm chart repositories",
				LastTransitionTime: metav1.Now(),
			})
			return err
		}

		var helmRepository *openshiftAPI.HelmChartRepository
		for _, item := range helmRepositories.Items {
			if item.Name == clusterTemplate.Spec.HelmChartRef.Repository {
				helmRepository = &item
				break
			}
		}

		if helmRepository == nil {
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               v1alpha1.InstallSucceeded,
				Status:             metav1.ConditionFalse,
				Reason:             v1alpha1.HelmChartRepoErrReason,
				Message:            "Failed to find helm repository",
				LastTransitionTime: metav1.Now(),
			})
			return errors.New("could not find helm repository CR")
		}

		if err := r.HelmClient.InstallChart(
			ctx,
			r.Client,
			*helmRepository,
			clusterTemplate,
			*clusterTemplateInstance,
		); err != nil {
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               v1alpha1.InstallSucceeded,
				Status:             metav1.ConditionFalse,
				Reason:             v1alpha1.HelmChartInstallErrReason,
				Message:            "Failed to install helm chart",
				LastTransitionTime: metav1.Now(),
			})
			return err
		}
		meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.InstallSucceeded,
			Status:             metav1.ConditionFalse,
			Reason:             v1alpha1.HelmReleaseInstallingReason,
			Message:            "Installing helm release",
			LastTransitionTime: metav1.Now(),
		})
	}
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterStatus(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Get helm release for clustertemplateinstance", "name", clusterTemplateInstance.Name)
	release, err := r.HelmClient.GetRelease(clusterTemplateInstance.Name)

	if err != nil {
		return err
	}

	log.Info(
		"Find kubeconfig/kubeadmin secrets for clustertemplateinstance",
		"name",
		clusterTemplateInstance.Name,
	)

	provider, err := clusterprovider.GetClusterProvider(release, log)

	if err != nil {
		return err
	}

	ready := false
	status := ""
	if provider != nil {
		ready, status, err = provider.GetClusterStatus(ctx, r.Client, *clusterTemplateInstance)
		if err != nil {
			return err
		}
	}

	if provider == nil || ready {
		meta.SetStatusCondition(
			&clusterTemplateInstance.Status.Conditions,
			metav1.Condition{
				Type:               v1alpha1.InstallSucceeded,
				Status:             metav1.ConditionTrue,
				Reason:             v1alpha1.InstalledReason,
				Message:            status,
				LastTransitionTime: metav1.Now(),
			},
		)
	} else {
		meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.InstallSucceeded,
			Status:             metav1.ConditionFalse,
			Reason:             v1alpha1.HelmReleaseInstallingReason,
			Message:            status,
			LastTransitionTime: metav1.Now(),
		})
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterCredentials(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	condition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		v1alpha1.InstallSucceeded,
	)
	setupCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		v1alpha1.SetupSucceeded,
	)

	if condition.Status == metav1.ConditionTrue && setupCondition.Status == metav1.ConditionTrue {

		kubeconfigSecret := corev1.Secret{}

		if err := r.Client.Get(
			ctx,
			client.ObjectKey{
				Name:      clusterTemplateInstance.GetKubeconfigRef(),
				Namespace: clusterTemplateInstance.Namespace,
			},
			&kubeconfigSecret,
		); err != nil {
			return err
		}

		kubeconfig := api.Config{}
		if err := yaml.Unmarshal(kubeconfigSecret.Data["kubeconfig"], &kubeconfig); err != nil {
			return err
		}
		clusterTemplateInstance.Status.APIserverURL = kubeconfig.Clusters[0].Cluster.Server
		clusterTemplateInstance.Status.AdminPassword = &corev1.LocalObjectReference{
			Name: clusterTemplateInstance.GetKubeadminPassRef(),
		}
		clusterTemplateInstance.Status.Kubeconfig = &corev1.LocalObjectReference{
			Name: clusterTemplateInstance.GetKubeconfigRef(),
		}
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterSetup(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	clusterTemplate v1alpha1.ClusterTemplate,
) error {
	log := ctrl.LoggerFrom(ctx)
	installCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		v1alpha1.InstallSucceeded,
	)
	setupCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		v1alpha1.SetupSucceeded,
	)

	if installCondition.Status == metav1.ConditionTrue &&
		setupCondition.Status != metav1.ConditionTrue {

		if clusterTemplate.Spec.ClusterSetup == nil {
			currentTime := metav1.Now()
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               v1alpha1.SetupSucceeded,
				Status:             metav1.ConditionTrue,
				Reason:             v1alpha1.SetupSucceeded,
				Message:            "No cluster setup defined",
				LastTransitionTime: currentTime,
			})
			clusterTemplateInstance.Status.CompletionTime = &currentTime
			return nil
		} else {
			if setupCondition.Reason == v1alpha1.ClusterNotReadyReason {
				log.Info(
					"Create cluster setup tekton pipelines for clustertemplateinstance",
					"name",
					clusterTemplateInstance.Name,
				)
				if err := clustersetup.CreateSetupPipelines(
					ctx,
					log,
					r.Client,
					clusterTemplate,
					clusterTemplateInstance,
				); err != nil {
					meta.SetStatusCondition(
						&clusterTemplateInstance.Status.Conditions,
						metav1.Condition{
							Type:               v1alpha1.SetupSucceeded,
							Status:             metav1.ConditionFalse,
							Reason:             v1alpha1.ClusterSetupFailedReason,
							Message:            "Failed to create tekton pipeline",
							LastTransitionTime: metav1.Now(),
						},
					)
					return err
				}
				meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
					Type:               v1alpha1.SetupSucceeded,
					Status:             metav1.ConditionFalse,
					Reason:             v1alpha1.ClusterSetupStartedReason,
					Message:            "Tekton pipeline started",
					LastTransitionTime: metav1.Now(),
				})
			}

			log.Info(
				"reconcile setup jobs for clustertemplateinstance",
				"name",
				clusterTemplateInstance.Name,
			)
			pipelineRuns := &pipeline.PipelineRunList{}

			pipelineLabelReq, _ := labels.NewRequirement(
				clustersetup.ClusterSetupInstance,
				selection.Equals,
				[]string{clusterTemplateInstance.Name},
			)
			selector := labels.NewSelector().Add(*pipelineLabelReq)

			if err := r.Client.List(
				ctx,
				pipelineRuns,
				&client.ListOptions{
					LabelSelector: selector,
					Namespace:     clusterTemplateInstance.Namespace,
				},
			); err != nil {
				return err
			}

			if len(pipelineRuns.Items) == 0 {
				log.Info(
					"No tekton pipeline found for clustertemplateinstance",
					"name",
					clusterTemplateInstance.Name,
				)
				return nil
			}

			pipelineRun := pipelineRuns.Items[0]
			if pipelineRun.Name == "" {
				meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
					Type:               v1alpha1.SetupSucceeded,
					Status:             metav1.ConditionTrue,
					Reason:             "ClusterSetupSucceeded",
					Message:            "Cluster setup succeeded",
					LastTransitionTime: metav1.Now(),
				})
				return nil
			}
			clusterSetupStatus := v1alpha1.PipelineStatus{
				PipelineRef: pipelineRun.Name,
				Status:      "Running",
			}
			for i := range pipelineRun.Status.Conditions {
				if pipelineRun.Status.Conditions[i].Type == "Succeeded" {
					switch pipelineRun.Status.Conditions[i].Status {
					case corev1.ConditionTrue:
						clusterSetupStatus.Status = "Succeeded"
						meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
							Type:               v1alpha1.SetupSucceeded,
							Status:             metav1.ConditionTrue,
							Reason:             "ClusterSetupSucceeded",
							Message:            "Cluster setup succeeded",
							LastTransitionTime: metav1.Now(),
						})
					case corev1.ConditionFalse:
						clusterSetupStatus.Status = "Failed"
						meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
							Type:               v1alpha1.SetupSucceeded,
							Status:             metav1.ConditionFalse,
							Reason:             "ClusterSetupFailed",
							Message:            "Cluster setup failed",
							LastTransitionTime: metav1.Now(),
						})
					default:
						clusterSetupStatus.Status = "Running"
						meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
							Type:               v1alpha1.SetupSucceeded,
							Status:             metav1.ConditionFalse,
							Reason:             "ClusterSetupRunning",
							Message:            "Cluster setup is running",
							LastTransitionTime: metav1.Now(),
						})
					}
				}
			}

			taskRunStatuses := []v1alpha1.Task{}

			if pipelineRun.Status.PipelineSpec != nil {
				for _, task := range pipelineRun.Status.PipelineSpec.Tasks {
					taskRunStatus := v1alpha1.Task{
						Name:   task.Name,
						Status: v1alpha1.TaskPending,
					}

					for _, taskRun := range pipelineRun.Status.TaskRuns {
						if taskRun.PipelineTaskName == task.Name {
							for i := range taskRun.Status.Conditions {
								if taskRun.Status.Conditions[i].Type == "Succeeded" {
									switch taskRun.Status.Conditions[i].Status {
									case corev1.ConditionTrue:
										taskRunStatus.Status = v1alpha1.TaskSucceeded
									case corev1.ConditionFalse:
										taskRunStatus.Status = v1alpha1.TaskFailed
									default:
										taskRunStatus.Status = v1alpha1.TaskRunning
									}
								}
							}
						}
					}

					taskRunStatuses = append(taskRunStatuses, taskRunStatus)
				}
			}

			clusterSetupStatus.Tasks = taskRunStatuses
			clusterTemplateInstance.Status.ClusterSetup = &clusterSetupStatus
			clusterTemplateInstance.Status.CompletionTime = pipelineRun.Status.CompletionTime
			return nil

		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterTemplateInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterTemplateInstance{}).
		Complete(r)
}

func SetDefaultConditions(clusterInstance *v1alpha1.ClusterTemplateInstance) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.InstallSucceeded,
		Status:             metav1.ConditionFalse,
		Reason:             v1alpha1.HelmReleasePreparingReason,
		Message:            "Preparing helm install",
		LastTransitionTime: metav1.Now(),
	})
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.SetupSucceeded,
		Status:             metav1.ConditionFalse,
		Reason:             v1alpha1.ClusterNotReadyReason,
		Message:            "Waiting for cluster to be ready",
		LastTransitionTime: metav1.Now(),
	})
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.Ready,
		Status:             metav1.ConditionFalse,
		Reason:             v1alpha1.CreationInProgressReason,
		Message:            "Creation in progress",
		LastTransitionTime: metav1.Now(),
	})
}
