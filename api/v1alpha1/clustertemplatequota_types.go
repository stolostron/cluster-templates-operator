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

type AllowedTemplate struct {
	Name string `json:"name"`
	//+kubebuilder:validation:Minimum=1
	// +optional
	Count int `json:"count,omitempty"`
}

type ClusterTemplateQuotaSpec struct {
	//+kubebuilder:validation:Minimum=1
	// +optional
	Budget           int               `json:"budget,omitempty"`
	AllowedTemplates []AllowedTemplate `json:"allowedTemplates"`
}

// ClusterTemplateQuotaStatus defines the observed state of ClusterTemplateQuota
type ClusterTemplateQuotaStatus struct {
	BudgetSpent       int               `json:"budgetSpent"`
	TemplateInstances []AllowedTemplate `json:"templateInstances"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=clustertemplatequotas,shortName=ctq;ctqs,scope=Namespaced

// ClusterTemplateQuota is the Schema for the clustertemplatequota API
type ClusterTemplateQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterTemplateQuotaSpec   `json:"spec,omitempty"`
	Status ClusterTemplateQuotaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterTemplateQuotaList contains a list of ClusterTemplateQuota
type ClusterTemplateQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterTemplateQuota `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterTemplateQuota{}, &ClusterTemplateQuotaList{})
}
