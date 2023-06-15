package controllers

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	applicationset "github.com/argoproj/applicationset/pkg/utils"
	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/templates"
)

var defaultApplicationSets = map[string]*argo.ApplicationSet{
	"hypershift-cluster":          templates.HypershiftClusterAppSet,
	"hypershift-kubevirt-cluster": templates.HypershiftKubevirtClusterAppSet,
	"hypershift-agent-cluster":    templates.HypershiftAgentClusterAppSet,
}

var defaultTemplates = map[string]*v1alpha1.ClusterTemplate{
	"hypershift-cluster":          templates.HypershiftClusterCT,
	"hypershift-kubevirt-cluster": templates.HypershiftKubevirtClusterCT,
	"hypershift-agent-cluster":    templates.HypershiftAgentClusterCT,
}

type HypershiftTemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *HypershiftTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// A channel is used to generate an initial sync event.
	// Afterwards, the controller syncs on the Hypershift ClusterTemplate.
	initialSync := make(chan event.GenericEvent)
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterTemplate{}, builder.WithPredicates(predicate.NewPredicateFuncs(r.selectHypershiftTemplate))).
		Watches(&source.Channel{Source: initialSync}, &handler.EnqueueRequestForObject{}).
		Watches(
			&source.Kind{Type: &argo.ApplicationSet{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.NewPredicateFuncs(r.selectHypershiftTemplate)),
		).
		Complete(r); err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}
	go func() {
		for template := range defaultTemplates {
			initialSync <- event.GenericEvent{Object: defaultTemplates[template]}
		}
	}()
	return nil
}

// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets,verbs=get;list;watch;create;update

func (r *HypershiftTemplateReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	// Template
	defaultTemplate := defaultTemplates[req.NamespacedName.Name]
	template := &v1alpha1.ClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultTemplate.Name,
		},
	}
	if _, err := applicationset.CreateOrUpdate(ctx, r.Client, template, func() error {
		if !reflect.DeepEqual(template.Spec, defaultTemplate.Spec) || !reflect.DeepEqual(template.Labels, defaultTemplate.Labels) || !reflect.DeepEqual(template.Annotations, defaultTemplate.Annotations) {
			template.Spec = defaultTemplate.Spec
			template.Labels = defaultTemplate.Labels
			template.Annotations = defaultTemplate.Annotations
		}
		return nil
	}); err != nil {
		return reconcile.Result{}, err
	}

	// AppSet
	defaultAppSet := defaultApplicationSets[req.NamespacedName.Name]
	appSetTemplate := &argo.ApplicationSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultAppSet.Name,
			Namespace: ArgoCDNamespace,
		},
	}
	if _, err := applicationset.CreateOrUpdate(ctx, r.Client, appSetTemplate, func() error {
		// We need to re(set) generators only in case the appset don't exist:
		key := client.ObjectKeyFromObject(appSetTemplate)
		if err := r.Client.Get(ctx, key, appSetTemplate); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			appSetTemplate.Spec.Generators = []argo.ApplicationSetGenerator{{}}
		}

		if !reflect.DeepEqual(appSetTemplate.Spec.Template, defaultAppSet.Spec.Template) {
			appSetTemplate.Spec.Template = defaultAppSet.Spec.Template
		}
		return nil
	}); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *HypershiftTemplateReconciler) selectHypershiftTemplate(obj client.Object) bool {
	_, found := defaultTemplates[obj.GetName()]
	return found
}
