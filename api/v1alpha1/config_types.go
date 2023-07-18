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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ConfigSpec struct {
	// ArgoCd namespace where the ArgoCD instance is running
	ArgoCDNamespace string `json:"argoCDNamespace,omitempty"`
	// Custom UI image
	UIImage string `json:"uiImage,omitempty"`
	// Flag that indicate if UI console plugin should be deployed
	UIEnabled bool `json:"uiEnabled,omitempty"`
	// Override default timeout for logging into the new cluster. The default is set to 10 minutes
	// +optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^([0-9]+(\\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$"
	LoginAttemptTimeoutOverride *metav1.Duration `json:"loginAttemptTimeoutOverride,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=config,shortName=ctconfig;clustertemplateconfig,scope=Cluster
//+operator-sdk:csv:customresourcedefinitions:displayName="Configuration of cluster template",resources={{Pod, v1, ""}}

// Configuration of the cluster operator
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ConfigSpec `json:"spec"`
}

//+kubebuilder:object:root=true

// ConfigList contains a list of Config
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Config `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Config{}, &ConfigList{})
}
