package templates

import (
	_ "embed"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed hypershift-kubevirt-cluster-description.md
var hypershiftKubevirtClusterDescription string

var HypershiftKubevirtClusterCT = &v1alpha1.ClusterTemplate{
	ObjectMeta: metav1.ObjectMeta{
		Name: "hypershift-kubevirt-cluster",
		Labels: map[string]string{
			"clustertemplates.openshift.io/vendor": "community",
		},
		Annotations: map[string]string{
			"clustertemplates.openshift.io/description": hypershiftKubevirtClusterDescription,
		},
	},
	Spec: v1alpha1.ClusterTemplateSpec{
		Cost:              &cost,
		ClusterDefinition: "hypershift-kubevirt-cluster",
	},
}

var HypershiftKubevirtClusterAppSet = &argo.ApplicationSet{
	ObjectMeta: metav1.ObjectMeta{
		Name: "hypershift-kubevirt-cluster",
		Labels: map[string]string{
			"clustertemplates.openshift.io/vendor": "community",
		},
	},
	Spec: argo.ApplicationSetSpec{
		Generators: []argo.ApplicationSetGenerator{{}},
		Template: argo.ApplicationSetTemplate{
			Spec: argo.ApplicationSpec{
				Destination: argo.ApplicationDestination{
					Namespace: "clusters",
					Server:    "{{ url }}",
				},
				Project: "default",
				Source: &argo.ApplicationSource{
					RepoURL:        "https://stolostron.github.io/cluster-templates-manifests",
					TargetRevision: "0.0.3",
					Chart:          "hypershift-kubevirt-template",
				},
				SyncPolicy: &argo.SyncPolicy{
					Automated: &argo.SyncPolicyAutomated{},
				},
			},
		},
	},
}
