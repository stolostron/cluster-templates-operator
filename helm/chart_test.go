package helm

import (
	"context"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	"io/ioutil"
	"os"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	helmserver "github.com/stolostron/cluster-templates-operator/testutils/helm"
)

var _ = Describe("Helm client", func() {
	var server *httptest.Server
	BeforeEach(func() {
		server = helmserver.StartHelmRepoServer()
	})
	AfterEach(func() {
		server.Close()
	})
	It("GetChart", func() {
		helmClient := CreateHelmClient(k8sManager, cfg)
		chart, err := helmClient.GetChart(context.TODO(), "", "", "")
		Expect(chart).Should(BeNil())
		Expect(err).ShouldNot(BeNil())
		server := helmserver.StartHelmRepoServer()

		chart, err = helmClient.GetChart(context.TODO(), server.URL, "", "")
		Expect(chart).Should(BeNil())
		Expect(err).ShouldNot(BeNil())

		chart, err = helmClient.GetChart(context.TODO(), server.URL, "hypershift-template", "0.0.2")
		Expect(chart).ShouldNot(BeNil())
		Expect(err).Should(BeNil())
	})
})

func CreateHelmClient(k8sManager manager.Manager, config *rest.Config) *HelmClient {
	certDataFile, err := os.CreateTemp("", "certdata-*")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer certDataFile.Close()

	err = ioutil.WriteFile(certDataFile.Name(), config.CertData, 0644)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	certDataFileName := certDataFile.Name()

	keyDataFile, err := os.CreateTemp("", "keydata-*")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer keyDataFile.Close()

	err = ioutil.WriteFile(keyDataFile.Name(), config.KeyData, 0644)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	keyDataFileName := keyDataFile.Name()

	caDataFile, err := os.CreateTemp("", "cadata-*")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer caDataFile.Close()

	err = ioutil.WriteFile(caDataFile.Name(), config.CAData, 0644)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	caDataFileName := caDataFile.Name()

	return NewHelmClient(config, k8sManager.GetClient(), &certDataFileName, &keyDataFileName, &caDataFileName)
}
