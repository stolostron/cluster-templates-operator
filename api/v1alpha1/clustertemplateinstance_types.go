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
	"github.com/stolostron/cluster-templates-operator/argocd"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CTIFinalizer           = "clustertemplateinstance.openshift.io/finalizer"
	CTIRequesterAnnotation = "clustertemplates.openshift.io/requester"
	CTINameLabel           = "clustertemplateinstance.openshift.io/name"
	CTINamespaceLabel      = "clustertemplateinstance.openshift.io/namespace"
	CTISetupLabel          = "clustertemplate.openshift.io/cluster-setup"
	CTISetupSecretLabel    = "clustertemplate.openshift.io/cluster-setup-secret"
	CTRepoLabel            = "clustertemplate.openshift.io/repository"
)

type Parameter struct {
	// Name of the Helm parameter
	Name string `json:"name"`
	// Value of the Helm parameter
	Value string `json:"value"`
	// Name of the application set to which parameter is applied
	ApplicationSet string `json:"clusterSetup,omitempty"`
}

type ClusterTemplateInstanceSpec struct {
	// A reference to a secret which contains kubeconfig of the cluster. If specified day1 operation won't be executed.
	KubeconfigSecretRef *string `json:"kubeconfigSecretRef,omitempty"`
	// A reference to ClusterTemplate which will be used for installing and setting up the cluster
	ClusterTemplateRef string `json:"clusterTemplateRef"`
	// Helm parameters to be passed to cluster installation or setup
	Parameters []Parameter `json:"parameters,omitempty"`
}

type ClusterSetupStatus struct {
	// Name of the cluster setup
	Name string `json:"name"`
	// Status of the cluster setup
	Status argocd.ApplicationStatus `json:"status"`
	// Description of the cluster setup status
	Message string `json:"message"`
}

type Phase string

const (
	PendingPhase                    Phase  = "Pending"
	PendingMessage                  string = "Pending"
	ClusterDefinitionFailedPhase    Phase  = "ClusterDefinitionFailed"
	ClusterInstallingPhase          Phase  = "ClusterInstalling"
	ClusterInstallFailedPhase       Phase  = "ClusterInstallFailed"
	ManagedClusterFailedPhase       Phase  = "ManagedClusterFailed"
	ManagedClusterImportFailedPhase Phase  = "ManagedClusterImportFailed"
	KlusterletCreateFailedPhase     Phase  = "KlusterletCreateFailed"
	ArgoClusterFailedPhase          Phase  = "ArgoClusterFailed"
	AddingArgoClusterPhase          Phase  = "AddingArgoCluster"
	ClusterSetupCreateFailedPhase   Phase  = "ClusterSetupCreateFailedPhase"
	CreatingClusterSetupPhase       Phase  = "CreatingClusterSetup"
	ClusterSetupDegradedPhase       Phase  = "ClusterSetupDegradedPhase"
	ClusterSetupErrorPhase          Phase  = "ClusterSetupErrorPhase"
	ClusterSetupFailedPhase         Phase  = "ClusterSetupFailedPhase"
	ClusterSetupRunningPhase        Phase  = "ClusterSetupRunning"
	ReadyPhase                      Phase  = "Ready"
	CredentialsFailedPhase          Phase  = "CredentialsFailed"
	FailedPhase                     Phase  = "Failed"
)

type ClusterTemplateInstanceStatus struct {
	// A reference for secret which contains username and password under keys "username" and "password"
	// +operator-sdk:csv:customresourcedefinitions:type=status
	AdminPassword *corev1.LocalObjectReference `json:"adminPassword,omitempty"`
	// A reference for secret which contains kubeconfig under key "kubeconfig"
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Kubeconfig *corev1.LocalObjectReference `json:"kubeconfig,omitempty"`
	// API server URL of the new cluster
	// +operator-sdk:csv:customresourcedefinitions:type=status
	APIserverURL string `json:"apiServerURL,omitempty"`
	// Resource conditions
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions"`
	// Status of each cluster setup
	// +operator-sdk:csv:customresourcedefinitions:type=status
	ClusterSetup *[]ClusterSetupStatus `json:"clusterSetup,omitempty"`
	// Secrets create by cluster setup which provide credentials for applications created by cluster setup
	// +operator-sdk:csv:customresourcedefinitions:type=status
	ClusterSetupSecrets []corev1.LocalObjectReference `json:"clusterSetupSecrets,omitempty"`
	// A reference to ManagedCluster resource
	// +operator-sdk:csv:customresourcedefinitions:type=status
	ManagedCluster corev1.LocalObjectReference `json:"managedCluster,omitempty"`
	// Console URL of the new cluster. The value is taken from ManagedCluster.
	// +operator-sdk:csv:customresourcedefinitions:type=status
	ConsoleURL string `json:"consoleURL,omitempty"`
	// Represents instance installaton & setup phase
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Phase Phase `json:"phase"`
	// Additional message for Phase
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Message string `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=clustertemplateinstances,shortName=cti;ctis,scope=Namespaced
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Cluster phase"
//+kubebuilder:printcolumn:name="Adminpassword",type="string",JSONPath=".status.adminPassword.name",description="Admin Secret"
//+kubebuilder:printcolumn:name="Kubeconfig",type="string",JSONPath=".status.kubeconfig.name",description="Kubeconfig Secret"
//+kubebuilder:printcolumn:name="API URL",type="string",JSONPath=".status.apiServerURL",description="API URL"
//+operator-sdk:csv:customresourcedefinitions:displayName="Cluster template instance",resources={{Pod, v1, ""}}

// Represents instance of a cluster
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
