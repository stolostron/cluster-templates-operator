package controllers

import (
	"context"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/hashicorp/go-multierror"
	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/repository"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RepoEntry struct {
	Version string   `json:"version"`
	Urls    []string `json:"urls"`
}

type HelmRepo struct {
	Entries map[string][]RepoEntry `json:"entries"`
}

type ClusterTemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates,verbs=get
// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets,verbs=get;

func (r *ClusterTemplateReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	var errors *multierror.Error
	clusterTemplate := &v1alpha1.ClusterTemplate{}
	err := r.Get(ctx, req.NamespacedName, clusterTemplate)
	if err != nil {
		return ctrl.Result{}, err
	}

	appSet := &argo.ApplicationSet{}
	err = r.Get(
		ctx,
		types.NamespacedName{Name: clusterTemplate.Spec.ClusterDefinition, Namespace: ArgoCDNamespace},
		appSet,
	)
	if err != nil {
		errors = multierror.Append(errors, err)
		clusterTemplate.Status.ClusterDefinition.Error = pointer.String(err.Error())
	} else {
		clusterTemplate.Status.ClusterDefinition.Error = nil

		cdValues, cdParams, cdSchema, err := r.getValuesParamsAndSchema(
			ctx,
			appSet.Spec.Template.Spec,
		)
		if err != nil {
			errors = multierror.Append(errors, err)
			clusterTemplate.Status.ClusterDefinition.Error = pointer.String(err.Error())
		} else {
			clusterTemplate.Status.ClusterDefinition.Values = cdValues
			clusterTemplate.Status.ClusterDefinition.Params = cdParams
			clusterTemplate.Status.ClusterDefinition.Schema = cdSchema
			clusterTemplate.Status.ClusterDefinition.Error = nil
		}
	}

	clusterSetupStatus := []v1alpha1.ClusterSetupSchema{}
	for _, setup := range clusterTemplate.Spec.ClusterSetup {
		css := v1alpha1.ClusterSetupSchema{}
		css.Name = setup

		appSet := &argo.ApplicationSet{}
		err = r.Get(
			ctx,
			types.NamespacedName{Name: setup, Namespace: ArgoCDNamespace},
			appSet,
		)

		if err != nil {
			errors = multierror.Append(errors, err)
			css.Error = pointer.String(err.Error())
		} else {
			css.Error = nil

			values, params, schema, err := r.getValuesParamsAndSchema(
				ctx,
				appSet.Spec.Template.Spec,
			)
			if err != nil {
				errors = multierror.Append(errors, err)
				css.Error = pointer.String(err.Error())
			} else {
				css.Error = nil
				css.Values = values
				css.Params = params
				css.Schema = schema
			}
		}
		clusterSetupStatus = append(clusterSetupStatus, css)
	}
	clusterTemplate.Status.ClusterSetup = clusterSetupStatus

	err = r.Client.Status().Update(ctx, clusterTemplate)
	errors = multierror.Append(errors, err)
	return ctrl.Result{}, errors.ErrorOrNil()
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterTemplate{}).
		Complete(r)
}

func (r *ClusterTemplateReconciler) getValuesParamsAndSchema(
	ctx context.Context,
	appSpec argo.ApplicationSpec,
) (string, []v1alpha1.ClusterTemplateParams, string, error) {
	values := ""
	schema := ""
	params := []v1alpha1.ClusterTemplateParams{}

	if appSpec.Source.Helm != nil {
		for _, param := range appSpec.Source.Helm.Parameters {
			params = append(params, v1alpha1.ClusterTemplateParams{
				Name:  param.Name,
				Value: param.Value,
			})
		}
	}

	if appSpec.Source.Chart != "" {
		repoURL := appSpec.Source.RepoURL
		chartName := appSpec.Source.Chart
		chartVersion := appSpec.Source.TargetRevision
		chart, err := repository.GetChart(
			ctx,
			r.Client,
			repoURL,
			chartName,
			chartVersion,
			ArgoCDNamespace,
		)
		if err != nil {
			return values, params, schema, err
		}
		for _, file := range chart.Raw {
			if file.Name == "values.yaml" {
				values = string(file.Data)
			}
			if file.Name == "values.schema.json" {
				schema = string(file.Data)
			}
		}
		return values, params, schema, nil
	}
	return values, params, schema, nil
}
