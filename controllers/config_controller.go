package controllers

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	argo "github.com/argoproj-labs/argocd-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

const (
	argoCDNsConfig = "argocd-ns"
	enableUIConfig = "enable-ui"
	uiImageConfig  = "ui-image"

	defaultArgoCDNs      = "cluster-aas-operator"
	defaultDisableArgoCD = "false"
	defaultEnableUI      = "false"
	defaultUIImage       = "quay.io/stolostron/cluster-templates-console-plugin:latest"
)

var (
	ArgoCDNamespace      = defaultArgoCDNs
	EnableUI             = defaultEnableUI
	DisableArgo          = defaultDisableArgoCD
	UIImage              = defaultUIImage
	EnableUIconfigSync   = make(chan event.GenericEvent)
	EnableArgoconfigSync = make(chan event.GenericEvent)
)

type ConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *ConfigReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	config := &v1.ConfigMap{}
	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			ArgoCDNamespace = defaultArgoCDNs
			EnableUI = defaultEnableUI
			DisableArgo = defaultDisableArgoCD
			UIImage = defaultUIImage
			EnableUIconfigSync <- event.GenericEvent{Object: GetPluginDeployment()}
			EnableArgoconfigSync <- event.GenericEvent{Object: &argo.ArgoCD{ObjectMeta: metav1.ObjectMeta{Name: argoname, Namespace: ArgoCDNamespace}}}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	val, ok := config.Data[argoCDNsConfig]
	if ok && val != "" {
		ArgoCDNamespace = val
		EnableArgoconfigSync <- event.GenericEvent{Object: &argo.ArgoCD{ObjectMeta: metav1.ObjectMeta{Name: argoname, Namespace: ArgoCDNamespace}}}
	} else {
		ArgoCDNamespace = defaultArgoCDNs
		EnableArgoconfigSync <- event.GenericEvent{Object: &argo.ArgoCD{ObjectMeta: metav1.ObjectMeta{Name: argoname, Namespace: ArgoCDNamespace}}}
	}
	enableUI, enableUIOk := config.Data[enableUIConfig]
	uiImage, uiImageOk := config.Data[uiImageConfig]
	if enableUIOk || uiImageOk {
		EnableUI = enableUI
		if !uiImageOk {
			UIImage = defaultUIImage
		} else if uiImage != "" {
			UIImage = uiImage
		}
		EnableUIconfigSync <- event.GenericEvent{Object: GetPluginDeployment()}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&v1.ConfigMap{}, builder.WithPredicates(predicate.NewPredicateFuncs(selectCM))).
		Complete(r); err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}
	return nil
}

func selectCM(obj client.Object) bool {
	return obj.GetName() == "claas-config" && obj.GetNamespace() == "cluster-aas-operator"
}
