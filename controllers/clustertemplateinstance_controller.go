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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubernetes-client/go-base/config/api"
	clustertemplatev1alpha1 "github.com/rawagner/cluster-templates-operator/api/v1alpha1"

	"github.com/rawagner/cluster-templates-operator/helm"
	"github.com/rawagner/cluster-templates-operator/hypershift"
	"gopkg.in/yaml.v2"
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
	fmt.Println("Instance reconciler: start")
	clusterTemplateInstance := &clustertemplatev1alpha1.ClusterTemplateInstance{}
	err := r.Get(ctx, req.NamespacedName, clusterTemplateInstance)
	if err != nil {
		return ctrl.Result{}, err
	}

	if clusterTemplateInstance.GetDeletionTimestamp() != nil {
		fmt.Println("run delete finalizer")
		if controllerutil.ContainsFinalizer(clusterTemplateInstance, clusterTemplateInstanceFinalizer) {
			_, err = r.HelmClient.UninstallRelease(clusterTemplateInstance.Name)
			if err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(clusterTemplateInstance, clusterTemplateInstanceFinalizer)
			err := r.Update(ctx, clusterTemplateInstance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(clusterTemplateInstance, clusterTemplateInstanceFinalizer) {
		controllerutil.AddFinalizer(clusterTemplateInstance, clusterTemplateInstanceFinalizer)
		err = r.Update(ctx, clusterTemplateInstance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	fmt.Println("Instance reconciler: created ?")
	if !clusterTemplateInstance.Status.Created {
		fmt.Println("Instance reconciler: created no")
		values := make(map[string]interface{})
		err = json.Unmarshal([]byte(clusterTemplateInstance.Spec.Values), &values)
		if err != nil {
			return ctrl.Result{}, err
		}
		fmt.Println("Instance reconciler: call helm")
		err = r.HelmClient.InstallChart(
			clusterTemplateInstance.Spec.HelmRepositoryRef,
			clusterTemplateInstance.Name,
			values,
		)
		if err != nil {
			fmt.Println("err creating chart")
			return ctrl.Result{}, err
		}
		fmt.Println("Instance reconciler: call helm ok")
	}

	newStatus := clustertemplatev1alpha1.ClusterTemplateInstanceStatus{
		Created: true,
	}

	fmt.Println("Instance reconciler: get release")
	release, err := r.HelmClient.GetRelease(clusterTemplateInstance.Name)

	if err != nil {
		fmt.Println("err release info")
		return ctrl.Result{}, err
	}
	newStatus.Status = string(release.Info.Status)
	fmt.Println("Instance reconciler: get release ok")

	stringObjects := strings.Split(release.Manifest, "---\n")

	for _, obj := range stringObjects {
		var yamlObj map[string]interface{}
		err = yaml.Unmarshal([]byte(obj), &yamlObj)
		if err != nil {
			fmt.Println("ERR decoding yaml")
			return ctrl.Result{}, err
		}
		if yamlObj["kind"] == "HostedCluster" {

			hypershiftInfo, err := hypershift.GetHypershiftInfo(obj)

			fmt.Println("has hypershift info")
			if err != nil {
				return ctrl.Result{}, err
			}

			passSecret := v1.Secret{}
			err = r.Client.Get(
				ctx,
				client.ObjectKey{Name: hypershiftInfo.PassSecret, Namespace: hypershiftInfo.Namespace},
				&passSecret,
			)
			if err != nil {
				fmt.Println("pass secret not found")
			} else {
				newStatus.KubeadminPassword = string(passSecret.Data["password"])

				kubeconfigSecret := v1.Secret{}
				err = r.Client.Get(
					ctx,
					client.ObjectKey{Name: hypershiftInfo.ConfigSecret, Namespace: hypershiftInfo.Namespace},
					&kubeconfigSecret,
				)
				if err != nil {
					fmt.Println("kubeconfig not found")
				} else {
					kubeconfig := api.Config{}
					yaml.Unmarshal(kubeconfigSecret.Data["kubeconfig"], &kubeconfig)
					newStatus.APIserverURL = kubeconfig.Clusters[0].Cluster.Server
				}
			}
		}
	}

	fmt.Println("Instance reconciler: status updates")
	clusterTemplateInstance.Status = newStatus

	err = r.Status().Update(ctx, clusterTemplateInstance)

	if err != nil {
		fmt.Println("err instance status")
		return ctrl.Result{}, err
	}

	requeue := newStatus.APIserverURL == "" || newStatus.KubeadminPassword == ""

	return ctrl.Result{Requeue: requeue}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterTemplateInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustertemplatev1alpha1.ClusterTemplateInstance{}).
		Complete(r)
}
