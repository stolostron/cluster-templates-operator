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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterTemplateSetupSpec struct {
	// Skip the registeration of the cluster to the hub cluster
	SkipClusterRegistration bool `json:"skipClusterRegistration,omitempty"`

	// +optional
	// Array of ArgoCD applicationset names which are used for post installation setup of the cluster
	ClusterSetup []string `json:"clusterSetup,omitempty"`
}

type ClusterSetupSchema struct {
	// Name of the cluster setup step
	Name string `json:"name"`
	// Content of helm chart values.yaml
	Values string `json:"values,omitempty"`
	// Content of helm chart values.schema.json
	Schema string `json:"schema,omitempty"`
	// Contain information about failure during fetching helm chart
	// +optional
	Error *string `json:"error,omitempty"`
}

// ClusterTemplateStatus defines the observed state of ClusterTemplateSetup
type ClusterTemplateSetupStatus struct {
	// Describes helm chart properties and schema for every cluster setup step
	// +operator-sdk:csv:customresourcedefinitions:type=status
	ClusterSetup []ClusterSetupSchema `json:"clusterSetup,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=clustertemplatesetup,shortName=ctsetup;ctsetup,scope=Cluster
//+operator-sdk:csv:customresourcedefinitions:displayName="Cluster template setup",resources={{Pod, v1, ""}}

// Template of a cluster - post-install setup are defined as ArgoCD application set refs.
type ClusterTemplateSetup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterTemplateSetupSpec   `json:"spec"`
	Status ClusterTemplateSetupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterTemplateSetupList contains a list of ClusterTemplateSetup
type ClusterTemplateSetupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterTemplateSetup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterTemplateSetup{}, &ClusterTemplateSetupList{})
}
