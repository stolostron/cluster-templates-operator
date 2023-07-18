package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

const (
	ClusterDefinitionCreated ConditionType = "ClusterDefinitionCreated"
	ClusterInstallSucceeded  ConditionType = "ClusterInstallSucceeded"
	ManagedClusterCreated    ConditionType = "ManagedClusterCreated"
	ManagedClusterImported   ConditionType = "ManagedClusterImported"
	KlusterletAddonCreated   ConditionType = "KlusterletAddonCreated"
	ArgoClusterAdded         ConditionType = "ArgoClusterAdded"
	ClusterSetupCreated      ConditionType = "ClusterSetupCreated"
	ClusterSetupSucceeded    ConditionType = "ClusterSetupSucceeded"
	Ready                    ConditionType = "Ready"
	ConsoleURLRetrieved      ConditionType = "ConsoleURLRetrieved"
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
	ApplicationError               ClusterInstallReason = "ApplicationError"
	ClusterDefinitionNotCreated    ClusterInstallReason = "ClusterDefinitionNotCreated"
	ClusterProviderDetectionFailed ClusterInstallReason = "ClusterProviderDetectionFailed"
	ClusterStatusFailed            ClusterInstallReason = "ClusterStatusFailed"
	ClusterInstalled               ClusterInstallReason = "ClusterInstalled"
	ClusterInstalling              ClusterInstallReason = "ClusterInstalling"
)

type ConsoleURLReason string

const (
	ConsoleURLFailed    ConsoleURLReason = "ConsoleURLFailed"
	ConsoleURLSucceeded ConsoleURLReason = "ConsoleURLSucceeded"
	ConsoleURLSkipped   ConsoleURLReason = "ConsoleURLSkipped"
	ConsoleURLPending   ConsoleURLReason = "ConsoleURLPending"
)

type ManagedClusterCreatedReason string

const (
	MCFailed  ManagedClusterCreatedReason = "ManagedClusterFailed"
	MCCreated ManagedClusterCreatedReason = "ManagedClusterCreated"
	MCPending ManagedClusterCreatedReason = "ManagedClusterPending"
	MCSkipped ManagedClusterCreatedReason = "ManagedClusterSkipped"
)

type KlusterletCreatedReason string

const (
	KlusterletCreated      KlusterletCreatedReason = "KlusterletCreated"
	KlusterletFailed       KlusterletCreatedReason = "KlusterletFailed"
	KlusterletCreatePeding KlusterletCreatedReason = "KlusterletCreatePending"
	KlusterletSkipped      KlusterletCreatedReason = "KlusterletSkipped"
)

type ManagedClusterImportedReason string

const (
	MCImportFailed  ManagedClusterImportedReason = "ManagedClusterImportFailed"
	MCImported      ManagedClusterImportedReason = "ManagedClusterImported"
	MCImporting     ManagedClusterImportedReason = "ManagedClusterImporting"
	MCImportPending ManagedClusterImportedReason = "ManagedClusterImportPending"
	MCImportSkipped ManagedClusterImportedReason = "ManagedClusterImportSkipped"
)

type ArgoClusterAddedReason string

const (
	ArgoClusterFailed       ArgoClusterAddedReason = "ArgoClusterFailed"
	ArgoClusterCreated      ArgoClusterAddedReason = "ArgoClusterCreated"
	ArgoClusterPending      ArgoClusterAddedReason = "ArgoClusterPending"
	ArgoClusterLoginPending ArgoClusterAddedReason = "ArgoClusterLoginPending"
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
	ClusterSetupRunning      ClusterSetupSucceededReason = "ClusterSetupRunning"
	SetupSucceeded           ClusterSetupSucceededReason = "ClusterSetupSucceeded"
	ClusterSetupDegraded     ClusterSetupSucceededReason = "ClusterSetupDegraded"
	ClusterSetupError        ClusterSetupSucceededReason = "ClusterSetupError"
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

func (clusterInstance *ClusterTemplateInstance) SetConsoleURLCondition(
	status metav1.ConditionStatus,
	reason ConsoleURLReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(ConsoleURLRetrieved),
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

func (clusterInstance *ClusterTemplateInstance) SetManagedClusterCreatedCondition(
	status metav1.ConditionStatus,
	reason ManagedClusterCreatedReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(ManagedClusterCreated),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetManagedClusterImportedCondition(
	status metav1.ConditionStatus,
	reason ManagedClusterImportedReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(ManagedClusterImported),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetKlusterletCreatedCondition(
	status metav1.ConditionStatus,
	reason KlusterletCreatedReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(KlusterletAddonCreated),
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

func (clusterInstance *ClusterTemplateInstance) hasCondition(condition ConditionType) bool {
	return meta.FindStatusCondition(
		clusterInstance.Status.Conditions,
		string(condition),
	) != nil
}

func (clusterInstance *ClusterTemplateInstance) SetDefaultConditions() {
	if !clusterInstance.hasCondition(ClusterDefinitionCreated) {
		clusterInstance.SetClusterDefinitionCreatedCondition(
			metav1.ConditionFalse,
			ClusterDefinitionPending,
			"Pending",
		)
	}

	if !clusterInstance.hasCondition(ClusterInstallSucceeded) {
		clusterInstance.SetClusterInstallCondition(
			metav1.ConditionFalse,
			ClusterDefinitionNotCreated,
			"Waiting for cluster definition to be created",
		)
	}

	if !clusterInstance.hasCondition(ManagedClusterCreated) {
		clusterInstance.SetManagedClusterCreatedCondition(
			metav1.ConditionFalse,
			MCPending,
			"Waiting for cluster to be ready",
		)
	}

	if !clusterInstance.hasCondition(ManagedClusterImported) {
		clusterInstance.SetManagedClusterImportedCondition(
			metav1.ConditionFalse,
			MCImportPending,
			"Waiting for managed cluster to be created",
		)
	}

	if !clusterInstance.hasCondition(KlusterletAddonCreated) {
		clusterInstance.SetKlusterletCreatedCondition(
			metav1.ConditionFalse,
			KlusterletCreatePeding,
			"Waiting for managed cluster to be imported",
		)
	}

	if !clusterInstance.hasCondition(ArgoClusterAdded) {
		clusterInstance.SetArgoClusterAddedCondition(
			metav1.ConditionFalse,
			ArgoClusterPending,
			"Waiting for klusterlet to be created",
		)
	}

	if !clusterInstance.hasCondition(ClusterSetupCreated) {
		clusterInstance.SetClusterSetupCreatedCondition(
			metav1.ConditionFalse,
			ClusterNotInstalled,
			"Waiting for argo cluster to be created",
		)
	}

	if !clusterInstance.hasCondition(ClusterSetupSucceeded) {
		clusterInstance.SetClusterSetupSucceededCondition(
			metav1.ConditionFalse,
			ClusterSetupNotCreated,
			"Waiting for cluster setup to be created",
		)
	}

	if !clusterInstance.hasCondition(ConsoleURLRetrieved) {
		clusterInstance.SetConsoleURLCondition(
			metav1.ConditionFalse,
			ConsoleURLPending,
			"Pending",
		)
	}
}

func (cti *ClusterTemplateInstance) PhaseCanExecute(
	prevCondition ConditionType,
	currentCondition ...ConditionType,
) bool {
	condition := meta.FindStatusCondition(
		cti.Status.Conditions,
		string(prevCondition),
	)
	if condition.Status == metav1.ConditionFalse {
		return false
	}

	if len(currentCondition) == 0 {
		return true
	}
	condition = meta.FindStatusCondition(
		cti.Status.Conditions,
		string(currentCondition[0]),
	)
	return condition.Status != metav1.ConditionTrue
}
