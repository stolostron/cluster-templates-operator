package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

const (
	HelmChartInstallSucceeded ConditionType = "HelmChartInstallSucceeded"
	ClusterInstallSucceeded   ConditionType = "ClusterInstallSucceeded"
	SetupPipelineCreated      ConditionType = "SetupPipelineCreated"
	SetupPipelineSucceeded    ConditionType = "SetupPipelineSucceeded"
	Ready                     ConditionType = "Ready"
)

type HelmChartInstallReason string

const (
	HelmReleasePreparing  HelmChartInstallReason = "HelmReleasePreparing"
	HelmChartInstallError HelmChartInstallReason = "HelmChartInstallError"
	HelmChartInstalled    HelmChartInstallReason = "HelmChartInstalled"
	HelmChartNotSpecified HelmChartInstallReason = "HelmChartNotSpecified"
	HelmRepoListError     HelmChartInstallReason = "HelmRepoListError"
)

type ClusterInstallReason string

const (
	HelmReleaseNotInstalled        ClusterInstallReason = "HelmReleaseNotInstalled"
	HelmReleaseGetFailed           ClusterInstallReason = "HelmReleaseGetFailed"
	HelmReleaseNotFound            ClusterInstallReason = "HelmReleaseNotFound"
	ClusterProviderDetectionFailed ClusterInstallReason = "ClusterProviderDetectionFailed"
	ClusterStatusFailed            ClusterInstallReason = "ClusterStatusFailed"
	ClusterInstalled               ClusterInstallReason = "ClusterInstalled"
	ClusterInstalling              ClusterInstallReason = "ClusterInstalling"
)

type SetupPipelineCreatedReason string

const (
	ClusterNotInstalled    SetupPipelineCreatedReason = "ClusterNotInstalled"
	PipelineCreationFailed SetupPipelineCreatedReason = "PipelineCreationFailed"
	PipelineNotSpecified   SetupPipelineCreatedReason = "PipelineNotSpecified"
	PipelineCreated        SetupPipelineCreatedReason = "PipelineCreated"
)

type SetupPipelineReason string

const (
	PipelineRunNotCreated SetupPipelineReason = "PipelineRunNotCreated"
	PipelineNotDefined    SetupPipelineReason = "PipelineNotDefined"
	PipelineFetchFailed   SetupPipelineReason = "PipelineFetchFailed"
	PipelineNotFound      SetupPipelineReason = "PipelineNotFound"
	PipelineRunSucceeded  SetupPipelineReason = "PipelineRunSucceeded"
	PipelineRunFailed     SetupPipelineReason = "PipelineRunFailed"
	PipelineRunRunning    SetupPipelineReason = "PipelineRunRunning"
)

func (clusterInstance *ClusterTemplateInstance) SetHelmChartInstallCondition(
	status metav1.ConditionStatus,
	reason HelmChartInstallReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(HelmChartInstallSucceeded),
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

func (clusterInstance *ClusterTemplateInstance) SetSetupPipelineCreatedCondition(
	status metav1.ConditionStatus,
	reason SetupPipelineCreatedReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(SetupPipelineCreated),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetSetupPipelineCondition(
	status metav1.ConditionStatus,
	reason SetupPipelineReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(SetupPipelineSucceeded),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}
