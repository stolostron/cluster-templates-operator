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

// Package v1alpha1 contains API Schema definitions for the clustertemplate v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=clustertemplate.openshift.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "clustertemplate.openshift.io", Version: "v1alpha1"}

	APIVersion = GroupVersion.Group + "/" + GroupVersion.Version

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme

	HostedClusterGVK = schema.GroupVersionResource{
		Group:    "hypershift.openshift.io",
		Resource: "HostedCluster",
		Version:  "v1alpha1",
	}

	NodePoolGVK = schema.GroupVersionResource{
		Group:    "hypershift.openshift.io",
		Resource: "NodePool",
		Version:  "v1alpha1",
	}

	ClusterDeploymentGVK = schema.GroupVersionResource{
		Group:    "hive.openshift.io",
		Resource: "ClusterDeployment",
		Version:  "v1",
	}

	ClusterClaimGVK = schema.GroupVersionResource{
		Group:    "hive.openshift.io",
		Resource: "ClusterClaim",
		Version:  "v1",
	}

	ConsolePluginGVK = schema.GroupVersionResource{
		Group:    "console.openshift.io",
		Resource: "ConsolePlugin",
		Version:  "v1alpha1",
	}

	ManagedClusterGVK = schema.GroupVersionResource{
		Group:    "cluster.open-cluster-management.io",
		Resource: "ManagedCluster",
		Version:  "v1",
	}
)
