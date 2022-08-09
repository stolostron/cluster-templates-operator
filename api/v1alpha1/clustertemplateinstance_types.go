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
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterTemplateInstanceSpec defines the desired state of ClusterTemplateInstance
type ClusterTemplateInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ClusterTemplateInstance. Edit clustertemplateinstance_types.go to remove/update
	Template string `json:"template"`

	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	Values json.RawMessage `json:"values"`
}

type ClusterSetupStatus struct {
	Name           string             `json:"name"`
	Succeeded      v1.ConditionStatus `json:"succeeded"`
	Message        string             `json:"message"`
	Reason         string             `json:"reason"`
	CompletionTime *metav1.Time       `json:"completionTime,omitempty"`
}

type ClusterTemplateInstanceStatus struct {
	Created             bool                 `json:"created"`
	KubeadminPassword   string               `json:"kubeadminPassword"`
	APIserverURL        string               `json:"apiServerURL"`
	ClusterStatus       string               `json:"clusterStatus"`
	ClusterSetupStarted bool                 `json:"clusterSetupStarted"`
	ClusterSetup        []ClusterSetupStatus `json:"clusterSetup,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterTemplateInstance is the Schema for the clustertemplateinstances API
type ClusterTemplateInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterTemplateInstanceSpec   `json:"spec,omitempty"`
	Status ClusterTemplateInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterTemplateInstanceList contains a list of ClusterTemplateInstance
type ClusterTemplateInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterTemplateInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterTemplateInstance{}, &ClusterTemplateInstanceList{})
}
