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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	InstallSucceeded string = "InstallSucceeded"
	SetupSucceeded   string = "SetupSucceeded"
)

const (
	HelmReleasePreparingReason  string = "HelmReleasePreparing"
	ClusterNotReadyReason       string = "ClusterNotReady"
	ClusterSetupStartedReason   string = "ClusterSetupStarted"
	ClusterSetupFailedReason    string = "ClusterSetupFailed"
	InstalledReason             string = "Installed"
	HelmReleaseInstallingReason string = "HelmReleaseInstalling"
	HelmChartInstallErrReason   string = "HelmChartInstallErr"
	HelmChartRepoErrReason      string = "HelmChartRepoErr"
	HelmReleaseValuesErrReason  string = "HelmReleaseValuesErr"
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

type ClusterTemplateInstanceStatus struct {
	KubeadminPassword string             `json:"kubeadminPassword,omitempty"`
	Kubeconfig        string             `json:"kubeconfig,omitempty"`
	APIserverURL      string             `json:"apiServerURL,omitempty"`
	Conditions        []metav1.Condition `json:"conditions"`
	CompletionTime    *metav1.Time       `json:"completionTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=clustertemplateinstances,shortName=cti;ctis,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Install succeeded",type="string",JSONPath=".status.conditions[?(@.type==\"InstallSucceeded\")].status",description="Cluster installed"
//+kubebuilder:printcolumn:name="Setup succeeded",type="string",JSONPath=".status.conditions[?(@.type==\"SetupSucceeded\")].status",description="Cluster setup"
//+kubebuilder:printcolumn:name="Kubeadmin",type="string",JSONPath=".status.kubeadminPassword",description="Kubeadmin Secret"
//+kubebuilder:printcolumn:name="Kubeconfig",type="string",JSONPath=".status.kubeconfig",description="Kubeconfig Secret"
//+kubebuilder:printcolumn:name="API URL",type="string",JSONPath=".status.apiServerURL",description="API URL"

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
