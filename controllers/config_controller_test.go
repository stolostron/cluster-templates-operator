package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testutils "github.com/stolostron/cluster-templates-operator/testutils"
)

func createResource(obj client.Object) {
	err := k8sClient.Create(ctx, obj)
	Expect(err).ToNot(HaveOccurred())
	resourcesToDelete = append(resourcesToDelete, obj)
}

var resourcesToDelete []client.Object

var _ = Describe("CLaaS Config", func() {
	AfterEach(func() {
		for _, res := range resourcesToDelete {
			testutils.DeleteResource(ctx, res, k8sClient)
		}
		resourcesToDelete = []client.Object{}
	})
})
