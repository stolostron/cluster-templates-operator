package controllers

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	argo "github.com/argoproj-labs/argocd-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
)

const (
	configName = "config"

	defaultArgoCDNs         = "cluster-aas-operator"
	defaultEnableUI         = true
	defaultUIImage          = "quay.io/stolostron/cluster-templates-console-plugin:2.8.1-5ad79eb6b4d9533754364d19c6ef2b91e11807a7"
	argosyncNamePlaceholder = "~~argosync~~"
)

var (
	ArgoCDNamespace      = defaultArgoCDNs
	EnableUI             = false
	UIImage              = defaultUIImage
	EnableUIconfigSync   = make(chan event.GenericEvent)
	EnableArgoconfigSync = make(chan event.GenericEvent)
)

type ConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func getDefaultConfig() *v1alpha1.Config {
	return &v1alpha1.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configName,
			Namespace: defaultArgoCDNs,
		},
		Spec: v1alpha1.ConfigSpec{
			ArgoCDNamespace: defaultArgoCDNs,
			UIImage:         defaultUIImage,
			UIEnabled:       defaultEnableUI,
		},
	}
}

// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=config,verbs=get;list;watch;create;update;patch

func (r *ConfigReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	config := &v1alpha1.Config{}
	if err := r.Get(ctx, types.NamespacedName{Name: configName, Namespace: defaultArgoCDNs}, config); err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.Create(ctx, getDefaultConfig()); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if ArgoCDNamespace != config.Spec.ArgoCDNamespace {
		// Provision/Deprovison ArgoCD, recreate AppSets
		prevNs := ArgoCDNamespace
		ArgoCDNamespace = config.Spec.ArgoCDNamespace
		EnableArgoconfigSync <- event.GenericEvent{Object: &argo.ArgoCD{ObjectMeta: metav1.ObjectMeta{Name: argosyncNamePlaceholder, Namespace: prevNs}}}
	}

	if EnableUI != config.Spec.UIEnabled || UIImage != config.Spec.UIImage {
		EnableUI = config.Spec.UIEnabled
		UIImage = config.Spec.UIImage
		EnableUIconfigSync <- event.GenericEvent{Object: GetPluginDeployment()}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	initialSync := make(chan event.GenericEvent)
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Config{}).
		Watches(&source.Channel{Source: initialSync}, &handler.EnqueueRequestForObject{}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}
	go func() {
		initialSync <- event.GenericEvent{Object: getDefaultConfig()}
	}()
	return nil
}
