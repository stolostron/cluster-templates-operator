package controllers

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	argo "github.com/argoproj-labs/argocd-operator/api/v1alpha1"
	operators "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/utils"
)

const (
	argoname = "class-argocd"
)

type ArgoCDReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArgoCDReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// A channel is used to generate an initial sync event.
	initialSync := make(chan event.GenericEvent)
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&argo.ArgoCD{}, builder.WithPredicates(predicate.NewPredicateFuncs(selectArgo))).
		Watches(&source.Channel{Source: initialSync}, &handler.EnqueueRequestForObject{}).
		Watches(&source.Channel{Source: EnableArgoconfigSync}, &handler.EnqueueRequestForObject{}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}
	go func() {
		initialSync <- event.GenericEvent{Object: &argo.ArgoCD{ObjectMeta: metav1.ObjectMeta{Name: argoname, Namespace: defaultArgoCDNs}}}
	}()
	return nil
}

// +kubebuilder:rbac:groups=argoproj.io,resources=argocds,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions,verbs=get;list;watch;update;patch

func (r *ArgoCDReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	if ArgoCDNamespace == defaultArgoCDNs {
		if err := utils.EnsureResourceExists(ctx, r.Client, &argo.ArgoCD{
			ObjectMeta: metav1.ObjectMeta{
				Name:      argoname,
				Namespace: defaultArgoCDNs,
			},
			Spec: argo.ArgoCDSpec{
				Server: argo.ArgoCDServerSpec{
					Route: argo.ArgoCDRouteSpec{
						Enabled: true,
					},
				},
				ApplicationSet: &argo.ArgoCDApplicationSet{
					LogLevel: "info",
				},
			},
		}, false); err != nil {
			return reconcile.Result{}, err
		}

		// Add namespace to argo subscription namespaces:
		subscription, err := r.getSubscription(ctx)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Add env var to subscription:
		if subscription.Spec.Config == nil {
			subscription.Spec.Config = &operators.SubscriptionConfig{}
		}
		if !containsEnvVar(subscription) {
			subscription.Spec.Config.Env = append(subscription.Spec.Config.Env, corev1.EnvVar{
				Name:  "ARGOCD_CLUSTER_CONFIG_NAMESPACES",
				Value: defaultArgoCDNs,
			})
		} else {
			for i, env := range subscription.Spec.Config.Env {
				if env.Name == "ARGOCD_CLUSTER_CONFIG_NAMESPACES" {
					if !slices.Contains(strings.Split(env.Value, ","), defaultArgoCDNs) {
						if env.Value == "" {
							subscription.Spec.Config.Env[i].Value = defaultArgoCDNs
						} else {
							subscription.Spec.Config.Env[i].Value = fmt.Sprintf("%s,%s", env.Value, defaultArgoCDNs)
						}
					}
					break
				}
			}
		}

		if err := r.Update(ctx, subscription); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Remove ArgoCD instance only if on upstream
	if ArgoCDNamespace != defaultArgoCDNs {
		argocd := &argo.ArgoCD{}
		if err := r.Get(ctx, types.NamespacedName{Name: argoname, Namespace: defaultArgoCDNs}, argocd); err != nil {
			if !apierrors.IsNotFound(err) {
				return reconcile.Result{}, err
			}
		} else {
			if err := r.Client.Delete(ctx, argocd); err != nil {
				return reconcile.Result{}, err
			}
		}

		// Remove namespace from subscription
		subscription, err := r.getSubscription(ctx)
		if err != nil {
			return reconcile.Result{}, err
		}
		if subscription.Spec.Config == nil {
			return reconcile.Result{}, nil
		}

		newSlice := []corev1.EnvVar{}
		for _, n := range subscription.Spec.Config.Env {
			if n.Name != "ARGOCD_CLUSTER_CONFIG_NAMESPACES" {
				newSlice = append(newSlice, n)
			} else {
				values := strings.Split(n.Value, ",")
				index := slices.Index(values, defaultArgoCDNs)
				if index != -1 {
					newValues := append(values[:index], values[index+1:]...)
					n.Value = strings.Join(newValues, ",")
				}
				newSlice = append(newSlice, n)
			}
		}

		subscription.Spec.Config.Env = newSlice

		if err := r.Update(ctx, subscription); err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func containsEnvVar(subscription *operators.Subscription) bool {
	for _, env := range subscription.Spec.Config.Env {
		if env.Name == "ARGOCD_CLUSTER_CONFIG_NAMESPACES" {
			return true
		}
	}
	return false
}

func (r *ArgoCDReconciler) getSubscription(ctx context.Context) (*operators.Subscription, error) {
	subLabel, _ := labels.NewRequirement(
		fmt.Sprintf("%s/%s.%s", "operators.coreos.com", "argocd-operator", "openshift-operators"),
		selection.Exists,
		[]string{},
	)
	selector := labels.NewSelector().Add(*subLabel)

	subscriptions := &operators.SubscriptionList{}
	if err := r.List(
		ctx,
		subscriptions,
		&client.ListOptions{
			LabelSelector: selector,
			Namespace:     "openshift-operators",
		},
	); err != nil {
		return nil, err
	}

	if len(subscriptions.Items) == 0 {
		return nil, fmt.Errorf("subscription with argo label was not found")
	}
	subscription := subscriptions.Items[0]

	return &subscription, nil
}

func selectArgo(obj client.Object) bool {
	return obj.GetName() == argoname && obj.GetNamespace() == defaultArgoCDNs
}
