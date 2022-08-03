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
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"github.com/kubernetes-client/go-base/config/api"
	clustertemplatev1alpha1 "github.com/rawagner/cluster-templates-operator/api/v1alpha1"

	"github.com/rawagner/cluster-templates-operator/clustersetup"
	"github.com/rawagner/cluster-templates-operator/helm"
	"github.com/rawagner/cluster-templates-operator/hypershift"
	"gopkg.in/yaml.v2"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	clusterTemplate := clustertemplatev1alpha1.ClusterTemplate{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: clusterTemplateInstance.Spec.Template}, &clusterTemplate); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to fetch clustertemplate %q", err)
	}

	if err := r.reconcileClusterCreate(log, clusterTemplateInstance, clusterTemplate); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create cluster %q", err)
	}

	newStatus := &clustertemplatev1alpha1.ClusterTemplateInstanceStatus{
		Created:             true,
		ClusterSetupStarted: clusterTemplateInstance.Status.ClusterSetupStarted || false,
	}

	kubeconfigSecretName, err := r.reconcileClusterStatus(ctx, log, clusterTemplateInstance, newStatus)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile cluster status %q", err)
	}

	requeue, err := r.reconcileClusterSetup(ctx, log, clusterTemplateInstance, clusterTemplate, newStatus, kubeconfigSecretName)

	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile cluster setup %q", err)
	}
	return r.updateStatus(ctx, clusterTemplateInstance, newStatus, requeue)
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterCreate(
	log logr.Logger,
	clusterTemplateInstance *clustertemplatev1alpha1.ClusterTemplateInstance,
	clusterTemplate clustertemplatev1alpha1.ClusterTemplate,
) error {
	if !clusterTemplateInstance.Status.Created {
		log.Info("Create cluster from clustertemplateinstance", "name", clusterTemplateInstance.Name)
		values := make(map[string]interface{})
		err := json.Unmarshal(clusterTemplateInstance.Spec.Values, &values)
		if err != nil {
			return err
		}

		err = r.HelmClient.InstallChart(
			clusterTemplate.Spec.HelmChartURL,
			clusterTemplateInstance.Name,
			clusterTemplateInstance.Namespace,
			values,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterStatus(
	ctx context.Context,
	log logr.Logger,
	clusterTemplateInstance *clustertemplatev1alpha1.ClusterTemplateInstance,
	newStatus *clustertemplatev1alpha1.ClusterTemplateInstanceStatus,
) (string, error) {
	log.Info("Get helm release for clustertemplateinstance", "name", clusterTemplateInstance.Name)
	release, err := r.HelmClient.GetRelease(clusterTemplateInstance.Name)

	if err != nil {
		return "", err
	}
	newStatus.ClusterStatus = string(release.Info.Status)

	stringObjects := strings.Split(release.Manifest, "---\n")

	kubeconfigSecret := v1.Secret{}

	log.Info("Find kubeconfig/kubeadmin secrets for clustertemplateinstance", "name", clusterTemplateInstance.Name)
	for _, obj := range stringObjects {
		var yamlObj map[string]interface{}
		err = yaml.Unmarshal([]byte(obj), &yamlObj)
		if err != nil {
			return "", err
		}
		if yamlObj["kind"] == "HostedCluster" {
			log.Info("Get hypershift cluster info", "name", clusterTemplateInstance.Name)
			hypershiftInfo, status, err := hypershift.GetHypershiftInfo(ctx, obj, r.Client)
			newStatus.ClusterStatus = status
			if err != nil {
				return "", err
			}

			passSecret := v1.Secret{}
			err = r.Client.Get(
				ctx,
				client.ObjectKey{Name: hypershiftInfo.PassSecret, Namespace: hypershiftInfo.Namespace},
				&passSecret,
			)
			if err != nil {
				log.Info("pass secret not found", "name", clusterTemplateInstance.Name)
			} else {
				newStatus.KubeadminPassword = string(passSecret.Data["password"])

				err = r.Client.Get(
					ctx,
					client.ObjectKey{Name: hypershiftInfo.ConfigSecret, Namespace: hypershiftInfo.Namespace},
					&kubeconfigSecret,
				)
				if err != nil {
					log.Info("kubeconfig not found", "name", clusterTemplateInstance.Name)
				} else {
					kubeconfig := api.Config{}
					yaml.Unmarshal(kubeconfigSecret.Data["kubeconfig"], &kubeconfig)
					newStatus.APIserverURL = kubeconfig.Clusters[0].Cluster.Server
				}
			}
		}
	}
	return kubeconfigSecret.Name, nil
}

func (r *ClusterTemplateInstanceReconciler) reconcileClusterSetup(
	ctx context.Context,
	log logr.Logger,
	clusterTemplateInstance *clustertemplatev1alpha1.ClusterTemplateInstance,
	clusterTemplate clustertemplatev1alpha1.ClusterTemplate,
	newStatus *clustertemplatev1alpha1.ClusterTemplateInstanceStatus,
	kubeconfigSecret string,
) (bool, error) {
	if newStatus.ClusterStatus != "Available" {
		log.Info("cluster is not ready for setup yet", "name", clusterTemplateInstance.Name)
		return true, nil
	}
	if !newStatus.ClusterSetupStarted {
		log.Info("Create cluster setup jobs for clustertemplateinstance", "name", clusterTemplateInstance.Name)
		err := clustersetup.CreateSetupJobs(ctx, r.Client, clusterTemplate, clusterTemplateInstance, kubeconfigSecret)
		if err != nil {
			return true, err
		}
		newStatus.ClusterSetupStarted = true
		return true, nil
	}

	log.Info("reconcile setup jobs for clustertemplateinstance", "name", clusterTemplateInstance.Name)
	jobs := &batchv1.JobList{}

	jobLabelReq, _ := labels.NewRequirement("clusterinstance", selection.Equals, []string{clusterTemplateInstance.Name})
	selector := labels.NewSelector().Add(*jobLabelReq)

	err := r.Client.List(ctx, jobs, &client.ListOptions{LabelSelector: selector, Namespace: clusterTemplateInstance.Namespace})
	if err != nil {
		return true, err
	}
	newStatus.ClusterSetup = make([]clustertemplatev1alpha1.ClusterSetupStatus, 0)
	for _, job := range jobs.Items {
		setupStatus := clustertemplatev1alpha1.ClusterSetupStatus{}
		setupName := job.Labels["setupname"]
		if setupName != "" {
			setupStatus.Name = setupName

			completed := false
			for i := range job.Status.Conditions {
				if job.Status.Conditions[i].Type == "Complete" {
					completed = job.Status.Conditions[i].Status == "True"
				}
			}

			setupStatus.Completed = completed
			newStatus.ClusterSetup = append(newStatus.ClusterSetup, setupStatus)
		}
	}
	setupComplete := true
	for _, setupStatus := range newStatus.ClusterSetup {
		setupComplete = setupComplete && setupStatus.Completed
	}
	return !setupComplete, nil
}

func (r *ClusterTemplateInstanceReconciler) updateStatus(
	ctx context.Context,
	clusterTemplateInstance *clustertemplatev1alpha1.ClusterTemplateInstance,
	newStatus *clustertemplatev1alpha1.ClusterTemplateInstanceStatus,
	requeue bool,
) (ctrl.Result, error) {
	clusterTemplateInstance.Status = *newStatus

	err := r.Status().Update(ctx, clusterTemplateInstance)

	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: requeue, RequeueAfter: 60 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterTemplateInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustertemplatev1alpha1.ClusterTemplateInstance{}).
		Complete(r)
}
