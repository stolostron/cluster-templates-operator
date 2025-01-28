package templates

import (
	_ "embed"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// default cost of default templates
var cost int = 1

//go:embed hypershift-cluster-description.md
var hypershiftClusterDescription string

var HypershiftClusterCT = &v1alpha1.ClusterTemplate{
	ObjectMeta: metav1.ObjectMeta{
		Name: "hypershift-cluster",
		Labels: map[string]string{
			"clustertemplates.openshift.io/vendor": "community",
		},
		Annotations: map[string]string{
			"clustertemplates.openshift.io/description": hypershiftClusterDescription,
		},
	},
	Spec: v1alpha1.ClusterTemplateSpec{
		SkipClusterRegistration: true,
		Cost:                    &cost,
		ClusterDefinition:       "hypershift-cluster",
	},
}

var HypershiftClusterAppSet = &argo.ApplicationSet{
	ObjectMeta: metav1.ObjectMeta{
		Name: "hypershift-cluster",
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
					TargetRevision: "0.0.2",
					Chart:          "hypershift-template",
				},
				SyncPolicy: &argo.SyncPolicy{
					Automated: &argo.SyncPolicyAutomated{},
				},
			},
		},
	},
}
