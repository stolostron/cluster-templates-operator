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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kubernetes-client/go-base/config/api"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	clustertemplatev1alpha1 "github.com/rawagner/cluster-templates-operator/api/v1alpha1"

	"github.com/rawagner/cluster-templates-operator/clustersetup"
	"github.com/rawagner/cluster-templates-operator/helm"
	"github.com/rawagner/cluster-templates-operator/hypershift"
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

const clusterTemplateInstanceFinalizer = "clustertemplateinstance.rawagner.com/finalizer"

//+kubebuilder:rbac:groups=*,resources=*,verbs=*
// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterTemplateQuota object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ClusterTemplateInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	clusterTemplateInstance := &clustertemplatev1alpha1.ClusterTemplateInstance{}
	err := r.Get(ctx, req.NamespacedName, clusterTemplateInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("clustertemplateinstance not found, aborting reconcile", "name", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get clustertemplateinstance %q: %w", req.NamespacedName, err)
	}

	if clusterTemplateInstance.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(clusterTemplateInstance, clusterTemplateInstanceFinalizer) {
			_, err = r.HelmClient.UninstallRelease(clusterTemplateInstance.Name)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to uninstall clustertemplateinstance %q: %w", req.NamespacedName, err)
			}

			controllerutil.RemoveFinalizer(clusterTemplateInstance, clusterTemplateInstanceFinalizer)
			err := r.Update(ctx, clusterTemplateInstance)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer clustertemplateinstance %q: %w", req.NamespacedName, err)
			}
		}
		log.Info("Deleted clustertemplateinstance", "name", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(clusterTemplateInstance, clusterTemplateInstanceFinalizer) {
		controllerutil.AddFinalizer(clusterTemplateInstance, clusterTemplateInstanceFinalizer)
		err = r.Update(ctx, clusterTemplateInstance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if len(clusterTemplateInstance.Status.Conditions) == 0 {
		SetDefaultConditions(clusterTemplateInstance)
	}

	updErr := r.reconcile(ctx, log, clusterTemplateInstance)

	err = r.Status().Update(ctx, clusterTemplateInstance)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile cluster setup %q", err)
	}

	requeue := clusterTemplateInstance.Status.CompletionTime == nil
	return ctrl.Result{Requeue: requeue, RequeueAfter: 60 * time.Second}, updErr
}

func (r *ClusterTemplateInstanceReconciler) reconcile(
	ctx context.Context,
	log logr.Logger,
	clusterTemplateInstance *clustertemplatev1alpha1.ClusterTemplateInstance,
) error {
	clusterTemplate := clustertemplatev1alpha1.ClusterTemplate{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: clusterTemplateInstance.Spec.Template}, &clusterTemplate); err != nil {
		return fmt.Errorf("failed to fetch clustertemplate %q", err)
	}

	if err := r.reconcileClusterCreate(ctx, log, clusterTemplateInstance, clusterTemplate); err != nil {
		return fmt.Errorf("failed to create cluster %q", err)
	}

	kubeconfigSecretName, kubepassName, err := r.reconcileClusterStatus(ctx, log, clusterTemplateInstance)
	if err != nil {
		return fmt.Errorf("failed to reconcile cluster status %q", err)
	}

	if err := r.reconcileClusterSetup(ctx, log, clusterTemplateInstance, clusterTemplate, kubeconfigSecretName); err != nil {
		return fmt.Errorf("failed to reconcile cluster setup %q", err)
	}

	if err := r.reconcileClusterCredentials(ctx, log, clusterTemplateInstance, kubeconfigSecretName, kubepassName); err != nil {
		return fmt.Errorf("failed to reconcile cluster credentials %q", err)
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterCreate(
	ctx context.Context,
	log logr.Logger,
	clusterTemplateInstance *clustertemplatev1alpha1.ClusterTemplateInstance,
	clusterTemplate clustertemplatev1alpha1.ClusterTemplate,
) error {

	condition := meta.FindStatusCondition(clusterTemplateInstance.Status.Conditions, clustertemplatev1alpha1.InstallSucceeded)

	if condition.Status == metav1.ConditionFalse && condition.Reason != clustertemplatev1alpha1.HelmReleaseInstallingReason {
		log.Info("Create cluster from clustertemplateinstance", "name", clusterTemplateInstance.Name)

		values := make(map[string]interface{})
		err := json.Unmarshal(clusterTemplateInstance.Spec.Values, &values)
		if err != nil {
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               clustertemplatev1alpha1.InstallSucceeded,
				Status:             metav1.ConditionFalse,
				Reason:             clustertemplatev1alpha1.HelmReleaseValuesErrReason,
				Message:            "Failed to unmarshall helm chart values",
				LastTransitionTime: metav1.Now(),
			})
			return err
		}

		helmRepositories := &openshiftAPI.HelmChartRepositoryList{}
		err = r.Client.List(ctx, helmRepositories, &client.ListOptions{})
		if err != nil {
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               clustertemplatev1alpha1.InstallSucceeded,
				Status:             metav1.ConditionFalse,
				Reason:             clustertemplatev1alpha1.HelmChartRepoErrReason,
				Message:            "Failed to list helm chart repositories",
				LastTransitionTime: metav1.Now(),
			})
			return err
		}

		var helmRepository *openshiftAPI.HelmChartRepository
		for _, item := range helmRepositories.Items {
			if item.Name == clusterTemplate.Spec.HelmRepository {
				helmRepository = &item
				break
			}
		}

		if helmRepository == nil {
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               clustertemplatev1alpha1.InstallSucceeded,
				Status:             metav1.ConditionFalse,
				Reason:             clustertemplatev1alpha1.HelmChartRepoErrReason,
				Message:            "Failed to find helm repository",
				LastTransitionTime: metav1.Now(),
			})
			return errors.New("could not find helm repository CR")
		}

		helmChartURL, err := helm.GetChartURL(
			helmRepository.Spec.ConnectionConfig.URL,
			clusterTemplate.Spec.HelmChart,
			clusterTemplate.Spec.HelmChartVersion,
		)

		if err != nil {
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               clustertemplatev1alpha1.InstallSucceeded,
				Status:             metav1.ConditionFalse,
				Reason:             clustertemplatev1alpha1.HelmChartRepoErrReason,
				Message:            "Failed to get url of helm release",
				LastTransitionTime: metav1.Now(),
			})
			return err
		}

		err = r.HelmClient.InstallChart(
			helmChartURL,
			clusterTemplateInstance.Name,
			clusterTemplateInstance.Namespace,
			values,
		)
		if err != nil {
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               clustertemplatev1alpha1.InstallSucceeded,
				Status:             metav1.ConditionFalse,
				Reason:             clustertemplatev1alpha1.HelmChartInstallErrReason,
				Message:            "Failed to install helm chart",
				LastTransitionTime: metav1.Now(),
			})
			return err
		}
		meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
			Type:               clustertemplatev1alpha1.InstallSucceeded,
			Status:             metav1.ConditionFalse,
			Reason:             clustertemplatev1alpha1.HelmReleaseInstallingReason,
			Message:            "Installing helm release",
			LastTransitionTime: metav1.Now(),
		})
	}
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterStatus(
	ctx context.Context,
	log logr.Logger,
	clusterTemplateInstance *clustertemplatev1alpha1.ClusterTemplateInstance,
) (string, string, error) {
	log.Info("Get helm release for clustertemplateinstance", "name", clusterTemplateInstance.Name)
	release, err := r.HelmClient.GetRelease(clusterTemplateInstance.Name)

	if err != nil {
		return "", "", err
	}

	stringObjects := strings.Split(release.Manifest, "---\n")

	log.Info("Find kubeconfig/kubeadmin secrets for clustertemplateinstance", "name", clusterTemplateInstance.Name)
	for _, obj := range stringObjects {
		var yamlObj map[string]interface{}
		err = yaml.Unmarshal([]byte(obj), &yamlObj)
		if err != nil {
			return "", "", err
		}
		if yamlObj["kind"] == "HostedCluster" {
			log.Info("Get hypershift cluster info", "name", clusterTemplateInstance.Name)
			kubeSecret, ready, status, err := hypershift.GetHypershiftInfo(ctx, obj, r.Client)
			if err != nil {
				return "", "", err
			}

			if ready && kubeSecret.Kubeconfig != "" && kubeSecret.Kubeadmin != "" {
				meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
					Type:               clustertemplatev1alpha1.InstallSucceeded,
					Status:             metav1.ConditionTrue,
					Reason:             clustertemplatev1alpha1.InstalledReason,
					Message:            status,
					LastTransitionTime: metav1.Now(),
				})
			} else {
				meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
					Type:               clustertemplatev1alpha1.InstallSucceeded,
					Status:             metav1.ConditionFalse,
					Reason:             clustertemplatev1alpha1.HelmReleaseInstallingReason,
					Message:            status,
					LastTransitionTime: metav1.Now(),
				})
			}

			return kubeSecret.Kubeconfig, kubeSecret.Kubeadmin, nil
		}
	}
	return "", "", nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterCredentials(
	ctx context.Context,
	log logr.Logger,
	clusterTemplateInstance *clustertemplatev1alpha1.ClusterTemplateInstance,
	kubeconfigSecretName string,
	kubeadminSecretName string,
) error {
	condition := meta.FindStatusCondition(clusterTemplateInstance.Status.Conditions, clustertemplatev1alpha1.InstallSucceeded)
	setupCondition := meta.FindStatusCondition(clusterTemplateInstance.Status.Conditions, clustertemplatev1alpha1.SetupSucceeded)

	if condition.Status == metav1.ConditionTrue && condition.Reason == clustertemplatev1alpha1.InstalledReason &&
		setupCondition.Status == metav1.ConditionTrue {

		kubeconfigSecret := corev1.Secret{}

		err := r.Client.Get(
			ctx,
			client.ObjectKey{Name: kubeconfigSecretName, Namespace: clusterTemplateInstance.Namespace},
			&kubeconfigSecret,
		)
		if err != nil {
			return err
		}

		kubeconfig := api.Config{}
		yaml.Unmarshal(kubeconfigSecret.Data["kubeconfig"], &kubeconfig)
		clusterTemplateInstance.Status.APIserverURL = kubeconfig.Clusters[0].Cluster.Server
		clusterTemplateInstance.Status.KubeadminPassword = kubeadminSecretName
		clusterTemplateInstance.Status.Kubeconfig = kubeconfigSecretName
	}

	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterSetup(
	ctx context.Context,
	log logr.Logger,
	clusterTemplateInstance *clustertemplatev1alpha1.ClusterTemplateInstance,
	clusterTemplate clustertemplatev1alpha1.ClusterTemplate,
	kubeconfigSecret string,
) error {
	condition := meta.FindStatusCondition(clusterTemplateInstance.Status.Conditions, clustertemplatev1alpha1.InstallSucceeded)
	setupCondition := meta.FindStatusCondition(clusterTemplateInstance.Status.Conditions, clustertemplatev1alpha1.SetupSucceeded)

	if condition.Status == metav1.ConditionTrue &&
		setupCondition.Status != metav1.ConditionTrue &&
		kubeconfigSecret != "" {

		if setupCondition.Reason == clustertemplatev1alpha1.ClusterNotReadyReason {
			log.Info("Create cluster setup tekton pipelines for clustertemplateinstance", "name", clusterTemplateInstance.Name)
			err := clustersetup.CreateSetupPipelines(ctx, log, r.Client, clusterTemplate, clusterTemplateInstance, kubeconfigSecret)
			if err != nil {
				meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
					Type:               clustertemplatev1alpha1.SetupSucceeded,
					Status:             metav1.ConditionFalse,
					Reason:             clustertemplatev1alpha1.ClusterSetupFailedReason,
					Message:            "Failed to create tekton pipeline",
					LastTransitionTime: metav1.Now(),
				})
				return err
			}
			meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
				Type:               clustertemplatev1alpha1.SetupSucceeded,
				Status:             metav1.ConditionFalse,
				Reason:             clustertemplatev1alpha1.ClusterSetupStartedReason,
				Message:            "Tekton pipeline started",
				LastTransitionTime: metav1.Now(),
			})
		}

		log.Info("reconcile setup jobs for clustertemplateinstance", "name", clusterTemplateInstance.Name)
		pipelineRuns := &pipeline.PipelineRunList{}

		pipelineLabelReq, _ := labels.NewRequirement(clustersetup.ClusterSetupInstance, selection.Equals, []string{clusterTemplateInstance.Name})
		selector := labels.NewSelector().Add(*pipelineLabelReq)

		err := r.Client.List(ctx, pipelineRuns, &client.ListOptions{LabelSelector: selector, Namespace: clusterTemplateInstance.Namespace})
		if err != nil {
			return err
		}

		for _, pipelineRun := range pipelineRuns.Items {
			setupName := pipelineRun.Labels[clustersetup.ClusterSetupLabel]
			if setupName != "" {
				for i := range pipelineRun.Status.Conditions {
					if pipelineRun.Status.Conditions[i].Type == "Succeeded" {
						status := metav1.ConditionFalse
						if pipelineRun.Status.Conditions[i].Status == corev1.ConditionTrue {
							status = metav1.ConditionTrue
						}

						meta.SetStatusCondition(&clusterTemplateInstance.Status.Conditions, metav1.Condition{
							Type:               clustertemplatev1alpha1.SetupSucceeded,
							Status:             status,
							Reason:             pipelineRun.Status.Conditions[i].Reason,
							Message:            pipelineRun.Status.Conditions[i].Message,
							LastTransitionTime: metav1.Now(),
						})
					}
				}
				clusterTemplateInstance.Status.CompletionTime = pipelineRun.Status.CompletionTime
			}
		}
		return nil
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterTemplateInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustertemplatev1alpha1.ClusterTemplateInstance{}).
		Complete(r)
}

func SetDefaultConditions(clusterInstance *clustertemplatev1alpha1.ClusterTemplateInstance) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               clustertemplatev1alpha1.InstallSucceeded,
		Status:             metav1.ConditionFalse,
		Reason:             clustertemplatev1alpha1.HelmReleasePreparingReason,
		Message:            "Preparing helm install",
		LastTransitionTime: metav1.Now(),
	})
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               clustertemplatev1alpha1.SetupSucceeded,
		Status:             metav1.ConditionFalse,
		Reason:             clustertemplatev1alpha1.ClusterNotReadyReason,
		Message:            "Waiting for cluster to be ready",
		LastTransitionTime: metav1.Now(),
	})
}
