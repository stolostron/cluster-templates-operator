package argocd

import (
	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	argoHealth "github.com/argoproj/gitops-engine/pkg/health"
	"github.com/argoproj/gitops-engine/pkg/sync/common"
)

var _ = Describe("Application health", func() {
	It("Empty status", func() {
		app := &argo.Application{
			Status: argo.ApplicationStatus{},
		}
		status, msg := GetApplicationHealth(app)
		Expect(status).Should(Equal(ApplicationSyncRunning))
		Expect(msg).Should(Equal("Application sync is running"))
	})
	It("Has error condition", func() {
		app := &argo.Application{
			Status: argo.ApplicationStatus{
				Conditions: []argo.ApplicationCondition{
					{
						Type:    "fooError",
						Message: "foo msg",
					},
				},
			},
		}
		status, msg := GetApplicationHealth(app)
		Expect(status).Should(Equal(ApplicationError))
		Expect(msg).Should(Equal("foo msg"))
	})
	It("Health is degraded", func() {
		app := &argo.Application{
			Status: argo.ApplicationStatus{
				Health: argo.HealthStatus{
					Status: argoHealth.HealthStatusDegraded,
				},
			},
		}
		status, msg := GetApplicationHealth(app)
		Expect(status).Should(Equal(ApplicationDegraded))
		Expect(msg).Should(Equal("Application is degraded"))
	})
	It("Sync is running - operation is nil", func() {
		app := &argo.Application{
			Status: argo.ApplicationStatus{
				Health: argo.HealthStatus{
					Status: argoHealth.HealthStatusUnknown,
				},
			},
		}
		status, msg := GetApplicationHealth(app)
		Expect(status).Should(Equal(ApplicationSyncRunning))
		Expect(msg).Should(Equal("Application sync is running"))
	})
	It("Sync is running - operation is running", func() {
		app := &argo.Application{
			Status: argo.ApplicationStatus{
				Health: argo.HealthStatus{
					Status: argoHealth.HealthStatusHealthy,
				},
				OperationState: &argo.OperationState{
					Phase:   common.OperationRunning,
					Message: "foo msg",
				},
			},
		}
		status, msg := GetApplicationHealth(app)
		Expect(status).Should(Equal(ApplicationSyncRunning))
		Expect(msg).Should(Equal("foo msg"))
	})
	It("Synced", func() {
		app := &argo.Application{
			Status: argo.ApplicationStatus{
				Health: argo.HealthStatus{
					Status: argoHealth.HealthStatusHealthy,
				},
				OperationState: &argo.OperationState{
					Phase:   common.OperationSucceeded,
					Message: "foo msg",
				},
			},
		}
		status, msg := GetApplicationHealth(app)
		Expect(status).Should(Equal(ApplicationHealthy))
		Expect(msg).Should(Equal("Application is synced"))
	})
})
