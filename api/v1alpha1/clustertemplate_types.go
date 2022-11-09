/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterSetup struct {
	// Name of the cluster setup
	Name string `json:"name"`
	// ArgoCD application spec which is used for setting up the cluster
	Spec argo.ApplicationSpec `json:"spec"`
}

type ClusterTemplateSpec struct {
	// ArgoCD application spec which is used for installation of the cluster
	ClusterDefinition argo.ApplicationSpec `json:"clusterDefinition"`

	// +optional
	// Array of ArgoCD application specs which are used for post installation setup of the cluster
	ClusterSetup []ClusterSetup `json:"clusterSetup,omitempty"`

	//+kubebuilder:validation:Minimum=0
	// Cost of the cluster, used for quotas
	Cost int `json:"cost"`
}

// ClusterTemplateStatus defines the observed state of ClusterTemplate
type ClusterTemplateStatus struct {
	// Resource conditions
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=clustertemplates,shortName=ct;cts,scope=Cluster
//+kubebuilder:printcolumn:name="Cost",type="integer",JSONPath=".spec.cost",description="Cluster cost"
//+operator-sdk:csv:customresourcedefinitions:displayName="Cluster template",resources={{Pod, v1, ""}}

// Template of a cluster - defines resources required to start the installation (via Helm chart) and optionally a post install setup of the cluster (via Tekton pipeline)
type ClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterTemplateSpec   `json:"spec"`
	Status ClusterTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterTemplateList contains a list of ClusterTemplate
type ClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterTemplate{}, &ClusterTemplateList{})
}
