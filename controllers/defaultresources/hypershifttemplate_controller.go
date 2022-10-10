package defaultresources

import (
	"context"
	"encoding/json"
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

func getHypershiftTemplate() (*v1alpha1.ClusterTemplate, error) {
	defaultDomain, err := json.Marshal("sampletemplateinstance.com")
	if err != nil {
		return nil, err
	}
	defaultStr, err := json.Marshal("LoadBalancer")
	if err != nil {
		return nil, err
	}
	defaultVersion, err := json.Marshal("4.10.33")
	if err != nil {
		return nil, err
	}
	defaultArch, err := json.Marshal("x86_64")
	if err != nil {
		return nil, err
	}
	return &v1alpha1.ClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hypershift-cluster",
		},
		Spec: v1alpha1.ClusterTemplateSpec{
			Cost: 1,
			HelmChartRef: &v1alpha1.HelmChartRef{
				Repository: "cluster-templates",
				Name:       "hypershift-template",
				Version:    "0.0.1",
			},
			Properties: []v1alpha1.Property{
				{
					Name:         "baseDnsDomain",
					Description:  "Base DNS domain of the cluster",
					Type:         v1alpha1.PropertyTypeString,
					Overwritable: true,
					DefaultValue: defaultDomain,
				},
				{
					Name:         "APIPublishingStrategy",
					Description:  "API Publishing strategy - can be LoadBalancer or NodePort",
					Type:         v1alpha1.PropertyTypeString,
					Overwritable: true,
					DefaultValue: defaultStr,
				},
				{
					Name:         "ocpVersion",
					Description:  "OCP version to be used",
					Type:         v1alpha1.PropertyTypeString,
					Overwritable: true,
					DefaultValue: defaultVersion,
				},
				{
					Name:         "ocpArch",
					Description:  "OCP arch to be used",
					Type:         v1alpha1.PropertyTypeString,
					Overwritable: true,
					DefaultValue: defaultArch,
				},
				{
					Name:         "sshPublicKey",
					Description:  "SSH public key to be injected into all cluster node sshd servers",
					Type:         v1alpha1.PropertyTypeString,
					Overwritable: false,
					SecretRef: &v1alpha1.ResourceRef{
						Name:      "hypershift-cluster-secret",
						Namespace: "default",
					},
				},
				{
					Name:         "pullSecret",
					Description:  "Base64 encoded pull secret to be injected into the container runtime of all cluster nodes",
					Type:         v1alpha1.PropertyTypeString,
					Overwritable: false,
					SecretRef: &v1alpha1.ResourceRef{
						Name:      "hypershift-cluster-secret",
						Namespace: "default",
					},
				},
			},
		},
	}, nil
}

type HypershiftTemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *HypershiftTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	defaultTemplate, err := getHypershiftTemplate()
	if err != nil {
		return err
	}
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
		initialSync <- event.GenericEvent{Object: defaultTemplate}
	}()
	return nil
}

// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates,verbs=get;list;watch;create;update

func (r *HypershiftTemplateReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	defaultTemplate, err := getHypershiftTemplate()
	if err != nil {
		return ctrl.Result{}, err
	}
	template := &v1alpha1.ClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultTemplate.Name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, template, func() error {
		if !reflect.DeepEqual(template.Spec, defaultTemplate.Spec) {
			template.Spec = defaultTemplate.Spec
		}
		return nil
	})
	return reconcile.Result{}, err
}

func (r *HypershiftTemplateReconciler) selectHypershiftTemplate(obj client.Object) bool {
	defaultTemplate, err := getHypershiftTemplate()
	if err != nil {
		return false
	}
	return obj.GetName() == defaultTemplate.Name
}
