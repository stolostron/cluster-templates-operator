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
	"os"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
)

var (
	CLaaSlog            = logf.Log.WithName("claas-controller")
	ctiControllerCancel context.CancelFunc
)

type CLaaSReconciler struct {
	Manager ctrl.Manager
	client.Client
	enableHypershift     bool
	enableHive           bool
	enableConsolePlugin  bool
	enableManagedCluster bool
	enableKlusterlet     bool
}

// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

func (r *CLaaSReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	crd := &apiextensions.CustomResourceDefinition{}
	if err := r.Get(ctx, req.NamespacedName, crd); err != nil {
		if apierrors.IsNotFound(err) {
			CLaaSlog.Info("crd not found", "name", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//restart controller if needed

	if !r.enableHypershift && isCRDSupported(crd, v1alpha1.HostedClusterGVK) {
		r.enableHypershift = true
		ctiControllerCancel()
		ctiControllerCancel = StartCTIController(
			r.Manager,
			r.enableHypershift,
			r.enableHive,
			r.enableManagedCluster,
			r.enableKlusterlet,
		)

		if err := (&HypershiftTemplateReconciler{
			Client: r.Manager.GetClient(),
			Scheme: r.Manager.GetScheme(),
		}).SetupWithManager(r.Manager); err != nil {
			CLaaSlog.Error(err, "unable to create controller", "controller", "HypershiftTemplate")
			os.Exit(1)
		}
	}

	if !r.enableHive && isCRDSupported(crd, v1alpha1.ClusterDeploymentGVK) {
		r.enableHive = true
		ctiControllerCancel()
		ctiControllerCancel = StartCTIController(
			r.Manager,
			r.enableHypershift,
			r.enableHive,
			r.enableManagedCluster,
			r.enableKlusterlet,
		)
	}

	if !r.enableManagedCluster && isCRDSupported(crd, v1alpha1.ManagedClusterGVK) {
		r.enableManagedCluster = true
		ctiControllerCancel()
		ctiControllerCancel = StartCTIController(
			r.Manager,
			r.enableHypershift,
			r.enableHive,
			r.enableManagedCluster,
			r.enableKlusterlet,
		)
	}

	if !r.enableKlusterlet && isCRDSupported(crd, v1alpha1.KlusterletAddonGVK) {
		r.enableKlusterlet = true
		ctiControllerCancel()
		ctiControllerCancel = StartCTIController(
			r.Manager,
			r.enableHypershift,
			r.enableHive,
			r.enableManagedCluster,
			r.enableKlusterlet,
		)
	}

	if !r.enableConsolePlugin && isCRDSupported(crd, v1alpha1.ConsolePluginGVK) {
		r.enableConsolePlugin = true
		if err := (&ConsolePluginReconciler{
			Client: r.Manager.GetClient(),
			Scheme: r.Manager.GetScheme(),
		}).SetupWithManager(r.Manager); err != nil {
			CLaaSlog.Error(err, "unable to create controller", "controller", "ConsolePlugin")
			os.Exit(1)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CLaaSReconciler) SetupWithManager() error {
	client := r.Manager.GetClient()
	scheme := r.Manager.GetScheme()
	r.enableHypershift = isCRDAvailable(client, v1alpha1.HostedClusterGVK)
	r.enableHive = isCRDAvailable(client, v1alpha1.ClusterDeploymentGVK)
	r.enableConsolePlugin = isCRDAvailable(client, v1alpha1.ConsolePluginGVK)
	r.enableManagedCluster = isCRDAvailable(client, v1alpha1.ManagedClusterGVK)
	r.enableKlusterlet = isCRDAvailable(client, v1alpha1.KlusterletAddonGVK)

	ctiControllerCancel = StartCTIController(
		r.Manager,
		r.enableHypershift,
		r.enableHive,
		r.enableManagedCluster,
		r.enableKlusterlet,
	)

	if r.enableHypershift {
		if err := (&HypershiftTemplateReconciler{
			Client: client,
			Scheme: scheme,
		}).SetupWithManager(r.Manager); err != nil {
			CLaaSlog.Error(err, "unable to create controller", "controller", "HypershiftTemplate")
			os.Exit(1)
		}
	}

	if r.enableConsolePlugin {
		if err := (&ConsolePluginReconciler{
			Client: client,
			Scheme: scheme,
		}).SetupWithManager(r.Manager); err != nil {
			CLaaSlog.Error(err, "unable to create controller", "controller", "ConsolePlugin")
			os.Exit(1)
		}
	}

	return ctrl.NewControllerManagedBy(r.Manager).
		For(&apiextensions.CustomResourceDefinition{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(e event.DeleteEvent) bool {
					return false
				},
				CreateFunc: func(e event.CreateEvent) bool {
					return true
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					return false
				},
				GenericFunc: func(e event.GenericEvent) bool {
					return false
				},
			},
		).
		Complete(r)
}

func isCRDAvailable(client client.Client, gvk schema.GroupVersionResource) bool {
	_, err := client.RESTMapper().KindFor(gvk)

	found := err == nil
	if !found {
		CLaaSlog.Info(gvk.Resource + "CRD not found")
	}

	return found
}

func isCRDSupported(
	crd *apiextensions.CustomResourceDefinition,
	gvk schema.GroupVersionResource,
) bool {
	if crd.Spec.Group == gvk.Group && crd.Spec.Names.Kind == gvk.Resource {
		for _, version := range crd.Spec.Versions {
			if version.Name == gvk.Version {
				return true
			}
		}
	}
	return false
}
