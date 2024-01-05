package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

const (
	EnvironmentDefinitionCreated  ConditionType = "EnvironmentDefinitionCreated"
	EnvironmentInstallSucceeded   ConditionType = "EnvironmentInstallSucceeded"
	ManagedClusterCreated         ConditionType = "ManagedClusterCreated"
	ManagedClusterImported        ConditionType = "ManagedClusterImported"
	KlusterletAddonCreated        ConditionType = "KlusterletAddonCreated"
	ArgoClusterAdded              ConditionType = "ArgoClusterAdded"
	EnvironmentSetupCreated       ConditionType = "EnvironmentSetupCreated"
	EnvironmentSetupSucceeded     ConditionType = "EnvironmentSetupSucceeded"
	Ready                         ConditionType = "Ready"
	ConsoleURLRetrieved           ConditionType = "ConsoleURLRetrieved"
	NamespaceAccountCreated       ConditionType = "NamespaceAccountCreated"
	EnvironmentRBACSucceeded      ConditionType = "EnvironmentRBACSucceeded"
	NamespaceCredentialsSucceeded ConditionType = "NamespaceCredentialsSucceeded"
	AppLinksCollected             ConditionType = "AppLinksCollected"
)

type NamespaceCredentialsReason string

const (
	NamespaceCredentialsPending    NamespaceCredentialsReason = "NamespaceCredentialsPending"
	NamespaceCredentialsFailed     NamespaceCredentialsReason = "NamespaceCredentialsFailed"
	NamespaceCredentialsSuccceeded NamespaceCredentialsReason = "NamespaceCredentialsSuccceeded"
)

type EnvironmentDefinitionReason string

const (
	EnvironmentDefinitionPending EnvironmentDefinitionReason = "EnvironmentDefinitionPending"
	EnvironmentDefinitionFailed  EnvironmentDefinitionReason = "EnvironmentDefinitionFailed"
	ApplicationCreated           EnvironmentDefinitionReason = "ApplicationCreated"
)

type EnvironmentInstallReason string

const (
	ApplicationFetchFailed          EnvironmentInstallReason = "ApplicationFetchFailed"
	ApplicationDegraded             EnvironmentInstallReason = "ApplicationDegraded"
	ApplicationError                EnvironmentInstallReason = "ApplicationError"
	EnvironmentDefinitionNotCreated EnvironmentInstallReason = "EnvironmentDefinitionNotCreated"
	ClusterProviderDetectionFailed  EnvironmentInstallReason = "ClusterProviderDetectionFailed"
	ClusterStatusFailed             EnvironmentInstallReason = "ClusterStatusFailed"
	EnvironmentInstalled            EnvironmentInstallReason = "EnvironmentInstalled"
	EnvironmentInstalling           EnvironmentInstallReason = "EnvironmentInstalling"
)

type EnvironmentAccountReason string

const (
	EnvironmentAccountPending EnvironmentAccountReason = "EnvironmentAccountPending"
	EnvironmentAccountFailed  EnvironmentAccountReason = "EnvironmentAccountFailed"
	EnvironmentAccountCreated EnvironmentAccountReason = "EnvironmentAccountCreated"
)

type EnvironmentRBACReason string

const (
	EnvironmentRBACPending EnvironmentRBACReason = "EnvironmentRBACPending"
	EnvironmentRBACFailed  EnvironmentRBACReason = "EnvironmentRBACFailed"
	EnvironmentRBACCreated EnvironmentRBACReason = "EnvironmentRBACCreated"
)

type AppLinksReason string

const (
	AppLinksFailed    AppLinksReason = "AppLinksFailed"
	AppLinksSucceeded AppLinksReason = "AppLinksSucceeded"
	AppLinksPending   AppLinksReason = "AppLinksPending"
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

type EnvironmentSetupCreatedReason string

const (
	EnvironmentNotInstalled        EnvironmentSetupCreatedReason = "EnvironmentNotInstalled"
	EnvironmentSetupNotSpecified   EnvironmentSetupCreatedReason = "EnvironmentSetupNotSpecified"
	EnvironmentSetupCreationFailed EnvironmentSetupCreatedReason = "EnvironmentSetupCreationFailed"
	SetupCreated                   EnvironmentSetupCreatedReason = "EnvironmentSetupCreated"
)

type EnvironmentSetupSucceededReason string

const (
	EnvironmentSetupNotDefined   EnvironmentSetupSucceededReason = "EnvironmentSetupNotDefined"
	EnvironmentSetupFetchFailed  EnvironmentSetupSucceededReason = "EnvironmentSetupFetchFailed"
	EnvironmentSetupAppsNotFound EnvironmentSetupSucceededReason = "EnvironmentSetupAppsNotFound"
	EnvironmentSetupRunning      EnvironmentSetupSucceededReason = "EnvironmentSetupRunning"
	SetupSucceeded               EnvironmentSetupSucceededReason = "EnvironmentSetupSucceeded"
	EnvironmentSetupDegraded     EnvironmentSetupSucceededReason = "EnvironmentSetupDegraded"
	EnvironmentSetupError        EnvironmentSetupSucceededReason = "EnvironmentSetupError"
	EnvironmentSetupNotCreated   EnvironmentSetupSucceededReason = "EnvironmentSetupNotCreated"
)

func (clusterInstance *ClusterTemplateInstance) SetEnvironmentRBACCondition(
	status metav1.ConditionStatus,
	reason EnvironmentRBACReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(EnvironmentRBACSucceeded),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetEnvironmentAccountCondition(
	status metav1.ConditionStatus,
	reason EnvironmentAccountReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(NamespaceAccountCreated),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetNamespaceCredentialsCondition(
	status metav1.ConditionStatus,
	reason NamespaceCredentialsReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(NamespaceCredentialsSucceeded),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetEnvironmentDefinitionCreatedCondition(
	status metav1.ConditionStatus,
	reason EnvironmentDefinitionReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(EnvironmentDefinitionCreated),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetEnvironmentInstallCondition(
	status metav1.ConditionStatus,
	reason EnvironmentInstallReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(EnvironmentInstallSucceeded),
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

func (clusterInstance *ClusterTemplateInstance) SetAppLinksCollectedCondition(
	status metav1.ConditionStatus,
	reason AppLinksReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(AppLinksCollected),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func (clusterInstance *ClusterTemplateInstance) SetEnvironmentSetupCreatedCondition(
	status metav1.ConditionStatus,
	reason EnvironmentSetupCreatedReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(EnvironmentSetupCreated),
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

func (clusterInstance *ClusterTemplateInstance) SetEnvironmentSetupSucceededCondition(
	status metav1.ConditionStatus,
	reason EnvironmentSetupSucceededReason,
	message string,
) {
	meta.SetStatusCondition(&clusterInstance.Status.Conditions, metav1.Condition{
		Type:               string(EnvironmentSetupSucceeded),
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

func (clusterInstance *ClusterTemplateInstance) SetDefaultConditions(isNsType bool) {
	if isNsType {
		if !clusterInstance.hasCondition(EnvironmentDefinitionCreated) {
			clusterInstance.SetEnvironmentDefinitionCreatedCondition(
				metav1.ConditionFalse,
				EnvironmentDefinitionPending,
				"Pending",
			)
		}

		if !clusterInstance.hasCondition(EnvironmentInstallSucceeded) {
			clusterInstance.SetEnvironmentInstallCondition(
				metav1.ConditionFalse,
				EnvironmentDefinitionNotCreated,
				"Waiting for environment definition to be created",
			)
		}

		if !clusterInstance.hasCondition(NamespaceAccountCreated) {
			clusterInstance.SetEnvironmentAccountCondition(
				metav1.ConditionFalse,
				EnvironmentAccountPending,
				"Pending",
			)
		}

		if !clusterInstance.hasCondition(EnvironmentRBACSucceeded) {
			clusterInstance.SetEnvironmentRBACCondition(
				metav1.ConditionFalse,
				EnvironmentRBACPending,
				"Pending",
			)
		}

		if !clusterInstance.hasCondition(EnvironmentSetupCreated) {
			clusterInstance.SetEnvironmentSetupCreatedCondition(
				metav1.ConditionFalse,
				EnvironmentNotInstalled,
				"Waiting for argo cluster to be created",
			)
		}

		if !clusterInstance.hasCondition(EnvironmentSetupSucceeded) {
			clusterInstance.SetEnvironmentSetupSucceededCondition(
				metav1.ConditionFalse,
				EnvironmentSetupNotCreated,
				"Waiting for cluster setup to be created",
			)
		}

		if !clusterInstance.hasCondition(NamespaceCredentialsSucceeded) {
			clusterInstance.SetNamespaceCredentialsCondition(
				metav1.ConditionFalse,
				NamespaceCredentialsPending,
				"Waiting for cluster setup to finish",
			)
		}

		if !clusterInstance.hasCondition(AppLinksCollected) {
			clusterInstance.SetAppLinksCollectedCondition(
				metav1.ConditionFalse,
				AppLinksPending,
				"Pending",
			)
		}
	} else {
		if !clusterInstance.hasCondition(EnvironmentDefinitionCreated) {
			clusterInstance.SetEnvironmentDefinitionCreatedCondition(
				metav1.ConditionFalse,
				EnvironmentDefinitionPending,
				"Pending",
			)
		}

		if !clusterInstance.hasCondition(EnvironmentInstallSucceeded) {
			clusterInstance.SetEnvironmentInstallCondition(
				metav1.ConditionFalse,
				EnvironmentDefinitionNotCreated,
				"Waiting for environment definition to be created",
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

		if !clusterInstance.hasCondition(EnvironmentSetupCreated) {
			clusterInstance.SetEnvironmentSetupCreatedCondition(
				metav1.ConditionFalse,
				EnvironmentNotInstalled,
				"Waiting for argo cluster to be created",
			)
		}

		if !clusterInstance.hasCondition(EnvironmentSetupSucceeded) {
			clusterInstance.SetEnvironmentSetupSucceededCondition(
				metav1.ConditionFalse,
				EnvironmentSetupNotCreated,
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
