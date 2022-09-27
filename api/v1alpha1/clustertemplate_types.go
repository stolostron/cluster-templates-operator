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

	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ResourceRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ClusterSetup struct {
	// +optional
	PipelineRef pipeline.PipelineRef `json:"pipelineRef,omitempty"`

	// +optional
	Pipeline ResourceRef `json:"pipeline,omitempty"`
}

type HelmChartRef struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Repository string `json:"repository"`
}

type PropertyType string

const (
	String PropertyType = "string"
	Bool   PropertyType = "bool"
	Int    PropertyType = "int"
	Float  PropertyType = "float"
)

// TODO add admission webhook
type Property struct {
	Name string `json:"name"`

	Description string `json:"description"`

	// +kubebuilder:validation:Enum=string;bool;int;float
	Type PropertyType `json:"type"`

	Overwritable bool `json:"overwritable"`

	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	DefaultValue json.RawMessage `json:"defaultValue,omitempty"`

	// +optional
	SecretRef *ResourceRef `json:"secretRef,omitempty"`

	// +optional
	// prop Type can be string only
	ClusterSetup bool `json:"clusterSetup,omitempty"`
}

type ClusterTemplateSpec struct {
	HelmChartRef HelmChartRef `json:"helmChartRef"`

	// +optional
	ClusterSetup *ClusterSetup `json:"clusterSetup,omitempty"`

	//+kubebuilder:validation:Minimum=0
	Cost int `json:"cost"`

	// +optional
	Properties []Property `json:"properties,omitempty"`
}

// ClusterTemplateStatus defines the observed state of ClusterTemplate
type ClusterTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=clustertemplates,shortName=ct;cts,scope=Cluster
//+kubebuilder:printcolumn:name="Cost",type="integer",JSONPath=".spec.cost",description="Cluster cost"

// ClusterTemplate is the Schema for the clustertemplates API
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
