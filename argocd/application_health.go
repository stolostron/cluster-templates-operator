package argocd

import (
	"strings"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argoHealth "github.com/argoproj/gitops-engine/pkg/health"

	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
)

type ApplicationStatus string

const (
	ApplicationError       ApplicationStatus = "ApplicationError"
	ApplicationSyncRunning ApplicationStatus = "ApplicationSyncRunning"
	ApplicationDegraded    ApplicationStatus = "ApplicationDegraded"
	ApplicationHealthy     ApplicationStatus = "ApplicationHealthy"
)

// TODO ensure we actually need includeResourceHealth prop
func GetApplicationHealth(application *argo.Application, includeResourceHealth bool) (ApplicationStatus, string) {
	for _, condition := range application.Status.Conditions {
		if strings.HasSuffix(condition.Type, "Error") {
			return ApplicationError, condition.Message
		}
	}

	if application.Status.Health.Status == argoHealth.HealthStatusDegraded {
		msg := getOperationMsg(application)
		if msg == "" {
			msg = "Application is degraded"
		}
		return ApplicationDegraded, msg
	}

	if application.Status.OperationState == nil ||
		application.Status.OperationState.Phase != synccommon.OperationSucceeded ||
		application.Status.Health.Status != argoHealth.HealthStatusHealthy {

		msg := getOperationMsg(application)
		if msg == "" {
			msg = "Application sync is running"
		}

		return ApplicationSyncRunning, msg
	}

	if includeResourceHealth {
		for _, res := range application.Status.Resources {
			if res.Health == nil {
				if res.Status != argo.SyncStatusCodeSynced {
					return ApplicationSyncRunning, "Resource sync is running"
				}
			} else if res.Health.Status != argoHealth.HealthStatusHealthy {
				return ApplicationSyncRunning, "Resource sync is running"
			}
		}
	}

	return ApplicationHealthy, "Application is synced"
}

func getOperationMsg(application *argo.Application) string {
	if application.Status.OperationState != nil &&
		application.Status.OperationState.Message != "" {
		return application.Status.OperationState.Message
	}
	return ""
}
