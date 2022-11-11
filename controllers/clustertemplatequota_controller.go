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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
)

// ClusterTemplateQuotaReconciler reconciles a ClusterTemplateQuota object
type ClusterTemplateQuotaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplatequotas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplatequotas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplateinstances,verbs=get;list;watch
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates,verbs=get;list;watch

func (r *ClusterTemplateQuotaReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	clusterTemplateQuota := &v1alpha1.ClusterTemplateQuota{}
	if err := r.Get(ctx, req.NamespacedName, clusterTemplateQuota); err != nil {
		return ctrl.Result{}, err
	}

	clusterTemplateInstanceList := &v1alpha1.ClusterTemplateInstanceList{}
	listOpts := []client.ListOption{
		client.InNamespace(req.NamespacedName.Namespace),
	}
	if err := r.List(ctx, clusterTemplateInstanceList, listOpts...); err != nil {
		return ctrl.Result{}, err
	}

	clusterTemplateList := &v1alpha1.ClusterTemplateList{}
	if err := r.List(ctx, clusterTemplateList, []client.ListOption{}...); err != nil {
		return ctrl.Result{}, err
	}

	currentInstances := []v1alpha1.AllowedTemplate{}
	currentConst := 0
	for _, template := range clusterTemplateQuota.Spec.AllowedTemplates {
		count := 0

		templateCost := 0
		for _, cTemplate := range clusterTemplateList.Items {
			if cTemplate.Name == template.Name {
				templateCost = cTemplate.Spec.Cost
			}
		}

		for _, instance := range clusterTemplateInstanceList.Items {
			if instance.Spec.ClusterTemplateRef == template.Name {
				count++
				currentConst += templateCost
			}
		}

		currentInstances = append(currentInstances, v1alpha1.AllowedTemplate{
			Name:  template.Name,
			Count: count,
		})
	}

	clusterTemplateQuota.Status = v1alpha1.ClusterTemplateQuotaStatus{
		BudgetSpent:       currentConst,
		TemplateInstances: currentInstances,
	}

	if err := r.Status().Update(ctx, clusterTemplateQuota); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterTemplateQuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {

	mapInstanceToQuota := func(instance client.Object) []reconcile.Request {
		quotas := &v1alpha1.ClusterTemplateQuotaList{}

		listOpts := []client.ListOption{
			client.InNamespace(instance.GetNamespace()),
		}

		if err := r.List(context.Background(), quotas, listOpts...); err != nil {
			return []reconcile.Request{}
		}

		reply := make([]reconcile.Request, 0, len(quotas.Items))
		for _, quota := range quotas.Items {
			if instance.GetNamespace() == quota.Namespace {
				reply = append(reply, reconcile.Request{NamespacedName: types.NamespacedName{
					Namespace: quota.Namespace,
					Name:      quota.Name,
				}})
			}
		}
		return reply
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterTemplateQuota{}).
		Watches(
			&source.Kind{Type: &v1alpha1.ClusterTemplateInstance{}},
			handler.EnqueueRequestsFromMapFunc(mapInstanceToQuota)).
		Complete(r)
}
