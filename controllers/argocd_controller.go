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
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/utils"
)

const (
	argoname     = "class-argocd"
	secretName   = "class-argocd-secret"
	argoCDEnvVar = "ARGOCD_CLUSTER_CONFIG_NAMESPACES"
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
	if err := r.ensureDefaultSecretExists(ctx); err != nil {
		return reconcile.Result{}, err
	}

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
				Name:  argoCDEnvVar,
				Value: defaultArgoCDNs,
			})
		} else {
			for i, env := range subscription.Spec.Config.Env {
				if env.Name == argoCDEnvVar {
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

		err = r.Update(ctx, subscription)
		return reconcile.Result{}, err
	}

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
		if !apierrors.IsNotFound(err) {
			return reconcile.Result{}, err
		} else {
			return reconcile.Result{}, nil
		}
	}
	if subscription.Spec.Config == nil {
		return reconcile.Result{}, nil
	}

	newSlice := []corev1.EnvVar{}
	for _, n := range subscription.Spec.Config.Env {
		if n.Name != argoCDEnvVar {
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

	err = r.Update(ctx, subscription)
	return reconcile.Result{}, err
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
	subscriptions := &operators.SubscriptionList{}
	if err := r.List(ctx, subscriptions); err != nil {
		return nil, err
	}

	for _, sub := range subscriptions.Items {
		if sub.Spec.Package == "argocd-operator" || sub.Spec.Package == "openshift-gitops-operator" {
			return &sub, nil
		}
	}

	return nil, fmt.Errorf("subscription with argo label was not found")
}

func selectArgo(obj client.Object) bool {
	return obj.GetName() == argoname && obj.GetNamespace() == defaultArgoCDNs
}

func (r *ArgoCDReconciler) ensureDefaultSecretExists(ctx context.Context) error {
	// Ensure secret exists:
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ArgoCDNamespace}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.Create(ctx, getDefaultSecret()); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Ensure secret is removed from other namespaces
	ctiNameLabelReq, _ := labels.NewRequirement(
		v1alpha1.CTRepoLabel,
		selection.Exists,
		[]string{},
	)
	selector := labels.NewSelector().Add(*ctiNameLabelReq)
	secretList := &corev1.SecretList{}

	if err := r.Client.List(ctx, secretList, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return err
	}

	for _, s := range secretList.Items {
		if s.Namespace != ArgoCDNamespace {
			if err := r.Client.Delete(ctx, &s); err != nil {
				return err
			}
		}
	}

	return nil
}

func getDefaultSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ArgoCDNamespace,
			Labels: map[string]string{
				v1alpha1.CTRepoLabel:                   "",
				"argocd.argoproj.io/secret-type":       "repository",
				"clustertemplates.openshift.io/vendor": "community",
			},
		},
		StringData: map[string]string{
			"name": "cluster-templates-manifests",
			"type": "helm",
			"url":  "https://stolostron.github.io/cluster-templates-manifests",
		},
		Type: corev1.SecretTypeOpaque,
	}
}
