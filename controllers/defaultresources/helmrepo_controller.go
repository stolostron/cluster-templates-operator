package defaultresources

import (
	"context"
	"fmt"
	"reflect"

	openshiftAPI "github.com/openshift/api/helm/v1beta1"
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

	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
)

func getHelmRepo() *openshiftAPI.HelmChartRepository {
	return &openshiftAPI.HelmChartRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-templates",
		},
		Spec: openshiftAPI.HelmChartRepositorySpec{
			DisplayName: "Cluster templates",
			ConnectionConfig: openshiftAPI.ConnectionConfig{
				URL: "https://stolostron.github.io/cluster-templates-operator",
			},
		},
	}
}

type HelmRepoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelmRepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// A channel is used to generate an initial sync event.
	// Afterwards, the controller syncs on the Hypershift ClusterTemplate.
	initialSync := make(chan event.GenericEvent)
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterTemplate{}, builder.WithPredicates(predicate.NewPredicateFuncs(r.selectHelmRepo))).
		Watches(&source.Channel{Source: initialSync}, &handler.EnqueueRequestForObject{}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}
	go func() {
		initialSync <- event.GenericEvent{Object: getHelmRepo()}
	}()
	return nil
}

// +kubebuilder:rbac:groups=helm.openshift.io,resources=helmchartrepositories,verbs=get;list;watch;create;update

func (r *HelmRepoReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	defaultRepo := getHelmRepo()
	repo := &openshiftAPI.HelmChartRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultRepo.Name,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, repo, func() error {
		if !reflect.DeepEqual(repo.Spec, defaultRepo.Spec) {
			repo.Spec = defaultRepo.Spec
		}
		return nil
	})
	return reconcile.Result{}, err
}

func (r *HelmRepoReconciler) selectHelmRepo(obj client.Object) bool {
	return obj.GetName() == getHelmRepo().Name
}
