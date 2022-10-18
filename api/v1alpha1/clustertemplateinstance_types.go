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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CTIFinalizer      = "clustertemplateinstance.openshift.io/finalizer"
	CTINameLabel      = "clustertemplateinstance.openshift.io/name"
	CTINamespaceLabel = "clustertemplateinstance.openshift.io/namespace"
	CTISetupLabel     = "clustertemplate.openshift.io/cluster-setup"
	ArgoNamespace     = "openshift-gitops" // TODO make configurable
)

type Parameter struct {
	Name         string `json:"name"`
	Value        string `json:"value"`
	ClusterSetup string `json:"clusterSetup,omitempty"`
}

type ClusterTemplateInstanceSpec struct {
	ClusterTemplateRef string      `json:"clusterTemplateRef"`
	Parameters         []Parameter `json:"parameters,omitempty"`
}

type ClusterSetupStatus struct {
	Name   string            `json:"name"`
	Status argo.HealthStatus `json:"status"`
}

type Phase string

const (
	PendingPhase                  Phase  = "Pending"
	PendingMessage                string = "Pending"
	ClusterDefinitionFailedPhase  Phase  = "ClusterDefinitionFailed"
	ClusterInstallingPhase        Phase  = "ClusterInstalling"
	ClusterInstallFailedPhase     Phase  = "ClusterInstallFailed"
	ArgoClusterFailedPhase        Phase  = "ArgoClusterFailed"
	AddingArgoClusterPhase        Phase  = "AddingArgoCluster"
	ClusterSetupCreateFailedPhase Phase  = "ClusterSetupCreateFailedPhase"
	CreatingClusterSetupPhase     Phase  = "CreatingClusterSetup"
	ClusterSetupFailedPhase       Phase  = "ClusterSetupFailedPhase"
	ClusterSetupRunningPhase      Phase  = "ClusterSetupRunning"
	ReadyPhase                    Phase  = "Ready"
	CredentialsFailedPhase        Phase  = "CredentialsFailed"
	FailedPhase                   Phase  = "Failed"
)

type ClusterTemplateInstanceStatus struct {
	// A reference for secret which contains username and password under keys "username" and "password"
	AdminPassword *corev1.LocalObjectReference `json:"adminPassword,omitempty"`
	// A reference for secret which contains kubeconfig under key "kubeconfig"
	Kubeconfig   *corev1.LocalObjectReference `json:"kubeconfig,omitempty"`
	APIserverURL string                       `json:"apiServerURL,omitempty"`
	Conditions   []metav1.Condition           `json:"conditions"`
	ClusterSetup *[]ClusterSetupStatus        `json:"clusterSetup,omitempty"`
	Phase        Phase                        `json:"phase"`
	Message      string                       `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=clustertemplateinstances,shortName=cti;ctis,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Cluster phase"
//+kubebuilder:printcolumn:name="Adminpassword",type="string",JSONPath=".status.adminPassword.name",description="Admin Secret"
//+kubebuilder:printcolumn:name="Kubeconfig",type="string",JSONPath=".status.kubeconfig.name",description="Kubeconfig Secret"
//+kubebuilder:printcolumn:name="API URL",type="string",JSONPath=".status.apiServerURL",description="API URL"

// Represents a request for instance of cluster
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
