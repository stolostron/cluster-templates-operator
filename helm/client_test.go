package helm

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helm client", func() {
	It("Create client", func() {
		helmClient := NewHelmClient(cfg, k8sClient, nil, nil, nil)
		Expect(helmClient).ShouldNot(BeNil())
		Expect(helmClient.config).Should(Equal(cfg))
		Expect(*helmClient.ConfigFlags.APIServer).Should(Equal(cfg.Host))
		Expect(*helmClient.ConfigFlags.BearerToken).Should(Equal(cfg.BearerToken))
		Expect(*helmClient.ConfigFlags.CAFile).Should(Equal(cfg.CAFile))

		certDataFileName := "foo"
		keyDataFileName := "bar"
		caDataFileName := "baz"
		helmClient = NewHelmClient(
			cfg,
			k8sClient,
			&certDataFileName,
			&keyDataFileName,
			&caDataFileName,
		)
		Expect(helmClient).ShouldNot(BeNil())
		Expect(helmClient.config).Should(Equal(cfg))
		Expect(*helmClient.ConfigFlags.APIServer).Should(Equal(cfg.Host))
		Expect(*helmClient.ConfigFlags.BearerToken).Should(Equal(cfg.BearerToken))
		Expect(*helmClient.ConfigFlags.CAFile).Should(Equal(caDataFileName))
		Expect(*helmClient.ConfigFlags.CertFile).Should(Equal(certDataFileName))
		Expect(*helmClient.ConfigFlags.KeyFile).Should(Equal(keyDataFileName))
	})
})
