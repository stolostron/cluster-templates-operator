package defaultresources

import (
	"context"
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1alpha1 "github.com/rawagner/cluster-templates-operator/api/v1alpha1"
)

func getHypershiftTemplate() *v1alpha1.ClusterTemplate {
	return &v1alpha1.ClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hypershift-cluster",
		},
		Spec: v1alpha1.ClusterTemplateSpec{
			Cost: 1,
			HelmChartRef: &v1alpha1.HelmChartRef{
				Repository: "cluster-templates",
				Name:       "hypershift-chart",
				Version:    "0.0.1",
			},
		},
	}
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
		Complete(r); err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}
	go func() {
		initialSync <- event.GenericEvent{Object: getHypershiftTemplate()}
	}()
	return nil
}

// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates,verbs=get;list;watch;create;update

func (r *HypershiftTemplateReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	defaultTemplate := getHypershiftTemplate()
	template := &v1alpha1.ClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultTemplate.Name,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, template, func() error {
		if !reflect.DeepEqual(template.Spec, defaultTemplate.Spec) {
			template.Spec = defaultTemplate.Spec
		}
		return nil
	})
	return reconcile.Result{}, err
}

func (r *HypershiftTemplateReconciler) selectHypershiftTemplate(obj client.Object) bool {
	return obj.GetName() == getHypershiftTemplate().Name
}
