package templates

import (
	_ "embed"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed hypershift-agent-cluster-description.md
var hypershiftAgentClusterDescription string

var HypershiftAgentClusterCT = &v1alpha1.ClusterTemplate{
	ObjectMeta: metav1.ObjectMeta{
		Name: "hypershift-agent-cluster",
		Labels: map[string]string{
			"clustertemplates.openshift.io/vendor": "community",
		},
		Annotations: map[string]string{
			"clustertemplates.openshift.io/description": hypershiftAgentClusterDescription,
		},
	},
	Spec: v1alpha1.ClusterTemplateSpec{
		Cost:              &cost,
		ClusterDefinition: "hypershift-agent-cluster",
		ClusterSetup:      []string{"day2-kafka"},
	},
}

var HypershiftAgentClusterAppSet = &argo.ApplicationSet{
	ObjectMeta: metav1.ObjectMeta{
		Name: "hypershift-agent-cluster",
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
					TargetRevision: "0.0.1",
					Chart:          "hypershift-agent-template",
				},
				SyncPolicy: &argo.SyncPolicy{
					Automated: &argo.SyncPolicyAutomated{},
				},
			},
		},
	},
}

var Day2AppSet = &argo.ApplicationSet{
	ObjectMeta: metav1.ObjectMeta{
		Name: "day2-kafka",
		Labels: map[string]string{
			"clustertemplates.openshift.io/vendor": "community",
		},
	},
	Spec: argo.ApplicationSetSpec{
		Generators: []argo.ApplicationSetGenerator{{}},
		Template: argo.ApplicationSetTemplate{
			Spec: argo.ApplicationSpec{
				Destination: argo.ApplicationDestination{
					Namespace: "kafka",
					Server:    "{{ url }}",
				},
				Project: "default",
				Source: &argo.ApplicationSource{
					Path:           "day2-gitops/kafka",
					RepoURL:        "https://github.com/stolostron/cluster-templates-manifests",
					TargetRevision: "main",
				},
				SyncPolicy: &argo.SyncPolicy{
					Automated:   &argo.SyncPolicyAutomated{},
					SyncOptions: argo.SyncOptions{}.AddOption("CreateNamespace=true"),
				},
			},
		},
	},
}
