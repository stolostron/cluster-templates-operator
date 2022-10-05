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
	"fmt"
	"time"

	"github.com/kubernetes-client/go-base/config/api"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
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

type ClusterTemplateInstanceReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	HelmClient     *helm.HelmClient
	RequeueTimeout time.Duration
}

const clusterTemplateInstanceFinalizer = "clustertemplateinstance.openshift.io/finalizer"

// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplateinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplateinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=hypershift.openshift.io,resources=*,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=hive.openshift.io,resources=*,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=helm.openshift.io,resources=helmchartrepositories,verbs=get;list;watch
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelines,verbs=get;list;watch
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;delete

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
				_, ok := err.(*helm.ReleaseNotFoundErr)
				if !ok {
					return ctrl.Result{}, fmt.Errorf(
						"failed to get helm release %q: %w",
						req.NamespacedName,
						err,
					)
				} else {
					log.Info(
						"Helm release does not exist",
						"name",
						req.NamespacedName,
					)
				}
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
		clusterTemplateInstance.Status.Phase = v1alpha1.PendingPhase
		clusterTemplateInstance.Status.Message = v1alpha1.PendingMessage
	}

	err := r.reconcile(ctx, clusterTemplateInstance)

	setupSucceededCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.SetupPipelineSucceeded),
	)

	if updErr := r.Status().Update(ctx, clusterTemplateInstance); updErr != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update status of clustertemplateinstance %q: %w",
			req.NamespacedName,
			updErr,
		)
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: setupSucceededCondition.Status == metav1.ConditionFalse, RequeueAfter: r.RequeueTimeout}, nil
}

func (r *ClusterTemplateInstanceReconciler) reconcile(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	clusterTemplate := v1alpha1.ClusterTemplate{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: clusterTemplateInstance.Spec.ClusterTemplateRef}, &clusterTemplate); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.FailedPhase
		errMsg := fmt.Sprintf("failed to fetch ClusterTemplate - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileHelmChart(ctx, clusterTemplateInstance, clusterTemplate); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.HelmChartInstallFailedPhase
		errMsg := fmt.Sprintf("failed to install helm chart - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}
	if err := r.reconcileClusterStatus(
		ctx,
		clusterTemplateInstance,
		clusterTemplate,
	); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterInstallFailedPhase
		errMsg := fmt.Sprintf("failed to reconcile cluster status - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileClusterSetupCreate(ctx, clusterTemplateInstance, clusterTemplate); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.SetupPipelineCreateFailedPhase
		errMsg := fmt.Sprintf("failed to create cluster setup pipeline - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileClusterSetup(ctx, clusterTemplateInstance, clusterTemplate); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.SetupPipelineFailedPhase
		errMsg := fmt.Sprintf("failed to reconcile cluster setup - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	if err := r.reconcileClusterCredentials(ctx, clusterTemplateInstance); err != nil {
		clusterTemplateInstance.Status.Phase = v1alpha1.CredentialsFailedPhase
		errMsg := fmt.Sprintf("failed to reconcile cluster credentials - %q", err)
		clusterTemplateInstance.Status.Message = errMsg
		return fmt.Errorf(errMsg)
	}

	setPhase(clusterTemplateInstance)
	return nil
}

func setPhase(clusterTemplateInstance *v1alpha1.ClusterTemplateInstance) {
	helmChartInstallCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.HelmChartInstallSucceeded),
	)
	if helmChartInstallCondition != nil && helmChartInstallCondition.Status == metav1.ConditionTrue {
		clusterTemplateInstance.Status.Phase = v1alpha1.ClusterInstallingPhase
		clusterTemplateInstance.Status.Message = "Cluster is installing"
	}

	installSucceededCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ClusterInstallSucceeded),
	)
	if installSucceededCondition != nil && installSucceededCondition.Status == metav1.ConditionTrue {
		clusterTemplateInstance.Status.Phase = v1alpha1.SetupPipelineCreatingPhase
		clusterTemplateInstance.Status.Message = "Creating cluster setup pipeline"
	}

	setupCreatedCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.SetupPipelineCreated),
	)
	if setupCreatedCondition != nil && setupCreatedCondition.Status == metav1.ConditionTrue {
		clusterTemplateInstance.Status.Phase = v1alpha1.SetupPipelineRunningPhase
		clusterTemplateInstance.Status.Message = "Cluster setup is running"
	}

	setupSucceededCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.SetupPipelineSucceeded),
	)
	if setupSucceededCondition != nil && setupSucceededCondition.Status == metav1.ConditionTrue {
		clusterTemplateInstance.Status.Phase = v1alpha1.ReadyPhase
		clusterTemplateInstance.Status.Message = "Cluster is ready"
	}
}

func (r *ClusterTemplateInstanceReconciler) reconcileHelmChart(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	clusterTemplate v1alpha1.ClusterTemplate,
) error {
	log := ctrl.LoggerFrom(ctx)
	helmChartInstallCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.HelmChartInstallSucceeded),
	)

	if helmChartInstallCondition.Status == metav1.ConditionTrue {
		return nil
	}

	if clusterTemplate.Spec.HelmChartRef == nil {
		if _, ok := clusterTemplate.Annotations[clusterprovider.ClusterProviderExperimentalAnnotation]; ok {
			clusterTemplateInstance.SetHelmChartInstallCondition(
				metav1.ConditionTrue,
				v1alpha1.HelmChartNotSpecified,
				"No helm chart defined for the cluster template",
			)
			return nil
		}
		clusterTemplateInstance.SetHelmChartInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.HelmChartNotSpecified,
			"No helm chart defined for the cluster template",
		)
		return nil
	}

	log.Info(
		"Get helm chart of clustertemplateinstance",
		"name",
		clusterTemplateInstance.Name,
	)

	helmRepository := &openshiftAPI.HelmChartRepository{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: clusterTemplate.Spec.HelmChartRef.Repository}, helmRepository); err != nil {
		clusterTemplateInstance.SetHelmChartInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.HelmRepoListError,
			fmt.Sprintf("Failed to get helm chart repository - %q", err),
		)
		return err
	}

	if err := r.HelmClient.InstallChart(
		ctx,
		*helmRepository,
		clusterTemplate,
		*clusterTemplateInstance,
	); err != nil {
		clusterTemplateInstance.SetHelmChartInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.HelmChartInstallError,
			fmt.Sprintf("Failed to install helm chart - %q", err),
		)
		return err
	}

	clusterTemplateInstance.SetHelmChartInstallCondition(
		metav1.ConditionTrue,
		v1alpha1.HelmChartInstalled,
		"Helm chart installed",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterStatus(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	clusterTemplate v1alpha1.ClusterTemplate,
) error {
	log := ctrl.LoggerFrom(ctx)
	helmChartInstallCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.HelmChartInstallSucceeded),
	)
	if helmChartInstallCondition.Status == metav1.ConditionFalse {
		return nil
	}

	if _, ok := clusterTemplate.Annotations[clusterprovider.ClusterProviderExperimentalAnnotation]; ok {
		log.Info("Experimental provider specified", "name", clusterTemplateInstance.Name)
		return nil
	}

	log.Info("Get helm release for clustertemplateinstance", "name", clusterTemplateInstance.Name)
	release, err := r.HelmClient.GetRelease(clusterTemplateInstance.Name)

	if err != nil {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.HelmReleaseGetFailed,
			fmt.Sprintf("Failed to get helm release - %q", err),
		)
		return err
	}
	if release == nil {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.HelmReleaseNotFound,
			fmt.Sprintf("Failed to find helm release - %q", err),
		)
		return nil
	}

	provider, err := clusterprovider.GetClusterProvider(release, log)

	if err != nil {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterProviderDetectionFailed,
			fmt.Sprintf("Failed to detect cluster provider - %q", err),
		)
		return err
	}

	if provider == nil {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionTrue,
			v1alpha1.ClusterInstalled,
			"Available",
		)
		return nil
	}

	ready, status, err := provider.GetClusterStatus(ctx, r.Client, *clusterTemplateInstance)
	if err != nil {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterStatusFailed,
			fmt.Sprintf("Failed to detect cluster status - %q", err),
		)
		return err
	}

	if ready {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionTrue,
			v1alpha1.ClusterInstalled,
			status,
		)
	} else {
		clusterTemplateInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			v1alpha1.ClusterInstalling,
			status,
		)
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterCredentials(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	setupSucceededCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.SetupPipelineSucceeded),
	)

	if setupSucceededCondition.Status == metav1.ConditionFalse {
		return nil
	}

	if clusterTemplateInstance.Status.APIserverURL == "" {
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
	}

	clusterTemplateInstance.Status.AdminPassword = &corev1.LocalObjectReference{
		Name: clusterTemplateInstance.GetKubeadminPassRef(),
	}
	clusterTemplateInstance.Status.Kubeconfig = &corev1.LocalObjectReference{
		Name: clusterTemplateInstance.GetKubeconfigRef(),
	}
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterSetupCreate(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	clusterTemplate v1alpha1.ClusterTemplate,
) error {
	log := ctrl.LoggerFrom(ctx)
	installSucceededCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.ClusterInstallSucceeded),
	)

	if installSucceededCondition.Status == metav1.ConditionFalse {
		return nil
	}

	if clusterTemplate.Spec.ClusterSetup == nil {
		clusterTemplateInstance.SetSetupPipelineCreatedCondition(
			metav1.ConditionTrue,
			v1alpha1.PipelineNotSpecified,
			"No pipeline specified",
		)
		return nil
	}

	pipelineCreatedCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.SetupPipelineCreated),
	)

	if pipelineCreatedCondition.Status == metav1.ConditionTrue {
		return nil
	}

	log.Info(
		"Create cluster setup tekton pipeline for clustertemplateinstance",
		"name",
		clusterTemplateInstance.Name,
	)
	if err := clustersetup.CreateSetupPipeline(
		ctx,
		log,
		r.Client,
		clusterTemplate,
		clusterTemplateInstance,
	); err != nil {
		clusterTemplateInstance.SetSetupPipelineCreatedCondition(
			metav1.ConditionFalse,
			v1alpha1.PipelineCreationFailed,
			fmt.Sprintf("Failed to create tekton pipeline - %q", err),
		)
		return err
	}
	clusterTemplateInstance.SetSetupPipelineCreatedCondition(
		metav1.ConditionTrue,
		v1alpha1.PipelineCreated,
		"Tekton pipeline created",
	)
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterSetup(
	ctx context.Context,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	clusterTemplate v1alpha1.ClusterTemplate,
) error {
	log := ctrl.LoggerFrom(ctx)

	setupPipelineCreatedCondition := meta.FindStatusCondition(
		clusterTemplateInstance.Status.Conditions,
		string(v1alpha1.SetupPipelineCreated),
	)

	if setupPipelineCreatedCondition.Status == metav1.ConditionFalse {
		return nil
	}

	if clusterTemplate.Spec.ClusterSetup == nil {
		clusterTemplateInstance.SetSetupPipelineCondition(
			metav1.ConditionTrue,
			v1alpha1.PipelineNotDefined,
			"No pipeline defined",
		)
		return nil
	}

	log.Info(
		"reconcile tekton pipelines for clustertemplateinstance",
		"name",
		clusterTemplateInstance.Name,
	)
	pipelineRuns := &pipeline.PipelineRunList{}

	pipelineLabelReq, _ := labels.NewRequirement(
		clustersetup.ClusterSetupInstanceLabel,
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
		clusterTemplateInstance.SetSetupPipelineCondition(
			metav1.ConditionFalse,
			v1alpha1.PipelineFetchFailed,
			fmt.Sprintf("Failed to list pipelines - %q", err),
		)
		return err
	}

	if len(pipelineRuns.Items) == 0 {
		clusterTemplateInstance.SetSetupPipelineCondition(
			metav1.ConditionFalse,
			v1alpha1.PipelineNotFound,
			"Failed to find pipeline",
		)
		return nil
	}

	pipelineRun := pipelineRuns.Items[0]
	clusterSetupStatus := v1alpha1.Pipeline{
		PipelineRef: pipelineRun.Name,
		Status:      v1alpha1.PipelineRunning,
	}
	for i := range pipelineRun.Status.Conditions {
		if pipelineRun.Status.Conditions[i].Type == "Succeeded" {
			switch pipelineRun.Status.Conditions[i].Status {
			case corev1.ConditionTrue:
				clusterSetupStatus.Status = v1alpha1.PipelineSucceeded
				clusterTemplateInstance.SetSetupPipelineCondition(
					metav1.ConditionTrue,
					v1alpha1.PipelineRunSucceeded,
					"Pipeline run succeeded",
				)
			case corev1.ConditionFalse:
				clusterSetupStatus.Status = v1alpha1.PipelineFailed
				clusterTemplateInstance.SetSetupPipelineCondition(
					metav1.ConditionFalse,
					v1alpha1.PipelineRunFailed,
					"Pipeline run failed",
				)
			default:
				clusterSetupStatus.Status = v1alpha1.PipelineRunning
				clusterTemplateInstance.SetSetupPipelineCondition(
					metav1.ConditionFalse,
					v1alpha1.PipelineRunRunning,
					"Pipeline run is running",
				)
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
				if taskRun != nil && taskRun.PipelineTaskName == task.Name {
					for _, condition := range taskRun.Status.Conditions {
						if condition.Type == "Succeeded" {
							switch condition.Status {
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
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterTemplateInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterTemplateInstance{}).
		Complete(r)
}

func SetDefaultConditions(clusterInstance *v1alpha1.ClusterTemplateInstance) {
	clusterInstance.SetHelmChartInstallCondition(metav1.ConditionFalse, v1alpha1.HelmReleasePreparing, "Installing helm release")
	clusterInstance.SetClusterInstallCondition(metav1.ConditionFalse, v1alpha1.HelmReleaseNotInstalled, "Waiting for helm release")
	clusterInstance.SetSetupPipelineCreatedCondition(metav1.ConditionFalse, v1alpha1.ClusterNotInstalled, "Waiting for cluster to be ready")
	clusterInstance.SetSetupPipelineCondition(metav1.ConditionFalse, v1alpha1.PipelineRunNotCreated, "Waiting for PipelineRun to be created")
}
