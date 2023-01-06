package controllers

import (
	"context"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/helm"
	"k8s.io/apimachinery/pkg/runtime"
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
	Scheme     *runtime.Scheme
	HelmClient *helm.HelmClient
}

// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clustertemplate.openshift.io,resources=clustertemplates,verbs=get

func (r *ClusterTemplateReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	clusterTemplate := &v1alpha1.ClusterTemplate{}
	err := r.Get(ctx, req.NamespacedName, clusterTemplate)
	if err != nil {
		return ctrl.Result{}, err
	}

	cdValues, cdSchema, err := r.getValuesAndSchema(
		ctx,
		clusterTemplate.Spec.ClusterDefinition,
	)
	if err != nil {
		return ctrl.Result{}, err
	}
	clusterTemplate.Status.ClusterDefinition.Values = cdValues
	clusterTemplate.Status.ClusterDefinition.Schema = cdSchema

	clusterSetupStatus := []v1alpha1.ClusterSetupSchema{}
	for _, setup := range clusterTemplate.Spec.ClusterSetup {
		values, schema, err := r.getValuesAndSchema(
			ctx,
			setup.Spec,
		)
		if err != nil {
			return ctrl.Result{}, err
		}
		if values != "" || schema != "" {
			clusterSetupStatus = append(clusterSetupStatus, v1alpha1.ClusterSetupSchema{
				Name:   setup.Name,
				Values: values,
				Schema: schema,
			})
		}
	}
	clusterTemplate.Status.ClusterSetup = clusterSetupStatus

	err = r.Client.Status().Update(ctx, clusterTemplate)
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterTemplate{}).
		Complete(r)
}

func (r *ClusterTemplateReconciler) getValuesAndSchema(
	ctx context.Context,
	appSpec argo.ApplicationSpec,
) (string, string, error) {
	values := ""
	schema := ""
	if appSpec.Source.Chart != "" {
		repoURL := appSpec.Source.RepoURL
		chartName := appSpec.Source.Chart
		chartVersion := appSpec.Source.TargetRevision
		chart, err := r.HelmClient.GetChart(
			ctx,
			r.Client,
			repoURL,
			chartName,
			chartVersion,
			ArgoCDNamespace,
		)
		if err != nil {
			return values, schema, err
		}
		for _, file := range chart.Raw {
			if file.Name == "values.yaml" {
				values = string(file.Data)
			}
			if file.Name == "values.schema.json" {
				schema = string(file.Data)
			}
		}
		return values, schema, nil
	}
	return values, schema, nil
}
