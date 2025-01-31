package v1alpha1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ClusterTemplateInstance conditions", func() {
	It("Sets default conditions", func() {
		cti := ClusterTemplateInstance{}
		cti.SetDefaultConditions(false)

		Expect(len(cti.Status.Conditions)).To(Equal(9))
	})

	It("Updates condition", func() {
		cti := ClusterTemplateInstance{}
		cti.SetDefaultConditions(false)

		cti.SetEnvironmentDefinitionCreatedCondition(
			metav1.ConditionTrue,
			EnvironmentDefinitionFailed,
			"foo",
		)
		testCondition(cti, EnvironmentDefinitionCreated, string(EnvironmentDefinitionFailed))

		cti.SetEnvironmentInstallCondition(
			metav1.ConditionTrue,
			ApplicationFetchFailed,
			"foo",
		)
		testCondition(cti, EnvironmentInstallSucceeded, string(ApplicationFetchFailed))

		cti.SetEnvironmentSetupCreatedCondition(
			metav1.ConditionTrue,
			EnvironmentNotInstalled,
			"foo",
		)
		testCondition(cti, EnvironmentSetupCreated, string(EnvironmentNotInstalled))

		cti.SetManagedClusterCreatedCondition(
			metav1.ConditionTrue,
			MCFailed,
			"foo",
		)
		testCondition(cti, ManagedClusterCreated, string(MCFailed))

		cti.SetManagedClusterImportedCondition(
			metav1.ConditionTrue,
			MCImportFailed,
			"foo",
		)
		testCondition(cti, ManagedClusterImported, string(MCImportFailed))

		cti.SetArgoClusterAddedCondition(
			metav1.ConditionTrue,
			ArgoClusterFailed,
			"foo",
		)
		testCondition(cti, ArgoClusterAdded, string(ArgoClusterFailed))

		cti.SetEnvironmentSetupSucceededCondition(
			metav1.ConditionTrue,
			EnvironmentSetupNotDefined,
			"foo",
		)
		testCondition(cti, EnvironmentSetupSucceeded, string(EnvironmentSetupNotDefined))

		cti.SetConsoleURLCondition(
			metav1.ConditionTrue,
			ConsoleURLFailed,
			"foo",
		)
		testCondition(cti, ConsoleURLRetrieved, string(ConsoleURLFailed))
	})

	It("PhaseCanExecute", func() {
		cti := ClusterTemplateInstance{
			Status: ClusterTemplateInstanceStatus{
				Conditions: []metav1.Condition{
					{
						Type:   string(EnvironmentInstallSucceeded),
						Status: metav1.ConditionTrue,
					},
					{
						Type:   string(ManagedClusterCreated),
						Status: metav1.ConditionFalse,
					},
				},
			},
		}
		Expect(
			cti.PhaseCanExecute(EnvironmentInstallSucceeded, ManagedClusterCreated),
		).To(BeTrue())

		cti = ClusterTemplateInstance{
			Status: ClusterTemplateInstanceStatus{
				Conditions: []metav1.Condition{
					{
						Type:   string(EnvironmentInstallSucceeded),
						Status: metav1.ConditionFalse,
					},
					{
						Type:   string(ManagedClusterCreated),
						Status: metav1.ConditionFalse,
					},
				},
			},
		}
		Expect(
			cti.PhaseCanExecute(EnvironmentInstallSucceeded, ManagedClusterCreated),
		).To(BeFalse())

		cti = ClusterTemplateInstance{
			Status: ClusterTemplateInstanceStatus{
				Conditions: []metav1.Condition{
					{
						Type:   string(EnvironmentInstallSucceeded),
						Status: metav1.ConditionTrue,
					},
					{
						Type:   string(ManagedClusterCreated),
						Status: metav1.ConditionTrue,
					},
				},
			},
		}
		Expect(
			cti.PhaseCanExecute(EnvironmentInstallSucceeded, ManagedClusterCreated),
		).To(BeFalse())

		cti = ClusterTemplateInstance{
			Status: ClusterTemplateInstanceStatus{
				Conditions: []metav1.Condition{
					{
						Type:   string(EnvironmentInstallSucceeded),
						Status: metav1.ConditionFalse,
					},
					{
						Type:   string(ManagedClusterCreated),
						Status: metav1.ConditionTrue,
					},
				},
			},
		}
		Expect(
			cti.PhaseCanExecute(EnvironmentInstallSucceeded, ManagedClusterCreated),
		).To(BeFalse())
	})
})

func testCondition(cti ClusterTemplateInstance, conditionType ConditionType, reason string) {
	cond := meta.FindStatusCondition(
		cti.Status.Conditions,
		string(conditionType),
	)

	Expect(cond.Message).To(Equal("foo"))
	Expect(cond.Status).To(Equal(metav1.ConditionTrue))
	Expect(cond.Reason).To(Equal(reason))
}
