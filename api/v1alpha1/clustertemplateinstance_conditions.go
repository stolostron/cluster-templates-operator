package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

const (
	ClusterDefinitionCreated ConditionType = "ClusterDefinitionCreated"
	ClusterInstallSucceeded  ConditionType = "ClusterInstallSucceeded"
	ArgoClusterAdded         ConditionType = "ArgoClusterAdded"
	ClusterSetupCreated      ConditionType = "ClusterSetupCreated"
	ClusterSetupSucceeded    ConditionType = "ClusterSetupSucceeded"
	Ready                    ConditionType = "Ready"
)

type ClusterDefinitionReason string

const (
	ClusterDefinitionPending ClusterDefinitionReason = "ClusterDefinitionPending"
	ClusterDefinitionFailed  ClusterDefinitionReason = "ClusterDefinitionFailed"
	ApplicationCreated       ClusterDefinitionReason = "ApplicationCreated"
)

type ClusterInstallReason string

const (
	ApplicationFetchFailed         ClusterInstallReason = "ApplicationFetchFailed"
	ApplicationDegraded            ClusterInstallReason = "ApplicationDegraded"
	ClusterDefinitionNotCreated    ClusterInstallReason = "ClusterDefinitionNotCreated"
	ClusterProviderDetectionFailed ClusterInstallReason = "ClusterProviderDetectionFailed"
	ClusterStatusFailed            ClusterInstallReason = "ClusterStatusFailed"
	ClusterInstalled               ClusterInstallReason = "ClusterInstalled"
	ClusterInstalling              ClusterInstallReason = "ClusterInstalling"
)

type ArgoClusterAddedReason string

const (
	ArgoClusterFailed  ArgoClusterAddedReason = "ArgoClusterFailed"
	ArgoClusterCreated ArgoClusterAddedReason = "ArgoClusterCreated"
	ArgoClusterPending ArgoClusterAddedReason = "ArgoClusterPending"
)

type ClusterSetupCreatedReason string

const (
	ClusterNotInstalled        ClusterSetupCreatedReason = "ClusterNotInstalled"
	ClusterSetupNotSpecified   ClusterSetupCreatedReason = "ClusterSetupNotSpecified"
	ClusterSetupCreationFailed ClusterSetupCreatedReason = "ClusterSetupCreationFailed"
	SetupCreated               ClusterSetupCreatedReason = "ClusterSetupCreated"
)

type ClusterSetupSucceededReason string

const (
	ClusterSetupNotDefined   ClusterSetupSucceededReason = "ClusterSetupNotDefined"
	ClusterSetupFetchFailed  ClusterSetupSucceededReason = "ClusterSetupFetchFailed"
	ClusterSetupAppsNotFound ClusterSetupSucceededReason = "ClusterSetupAppsNotFound"
	SetupSucceeded           ClusterSetupSucceededReason = "ClusterSetupSucceeded"
	ClusterSetupDegraded     ClusterSetupSucceededReason = "ClusterSetupDegraded"
	ClusterSetupNotCreated   ClusterSetupSucceededReason = "ClusterSetupNotCreated"
)

func (clusterInstance *ClusterTemplateInstance) SetClusterDefinitionCreatedCondition(
	status metav1.ConditionStatus,
	reason ClusterDefinitionReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(ClusterDefinitionCreated),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetClusterInstallCondition(
	status metav1.ConditionStatus,
	reason ClusterInstallReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(ClusterInstallSucceeded),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetClusterSetupCreatedCondition(
	status metav1.ConditionStatus,
	reason ClusterSetupCreatedReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(ClusterSetupCreated),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetArgoClusterAddedCondition(
	status metav1.ConditionStatus,
	reason ArgoClusterAddedReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(ArgoClusterAdded),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetClusterSetupSucceededCondition(
	status metav1.ConditionStatus,
	reason ClusterSetupSucceededReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(ClusterSetupSucceeded),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}
