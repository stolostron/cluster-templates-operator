package helm

import (
	"context"
	"net/http/httptest"
	"net/url"

	argoCommon "github.com/argoproj/argo-cd/v2/common"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	"io/ioutil"
	"os"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	helmserver "github.com/stolostron/cluster-templates-operator/testutils/helm"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Helm client", func() {
	var server *httptest.Server
	var httpsServer *httptest.Server
	BeforeEach(func() {
		server = helmserver.StartHelmRepoServer()
		httpsServer = helmserver.StartHttpsHelmRepoServer()
	})
	AfterEach(func() {
		server.Close()
		httpsServer.Close()
	})
	It("GetChart", func() {
		helmClient := CreateHelmClient(k8sManager, cfg)
		chart, err := helmClient.GetChart(context.TODO(), k8sClient, "", "", "", "argocd")
		Expect(chart).Should(BeNil())
		Expect(err).ShouldNot(BeNil())
		server := helmserver.StartHelmRepoServer()

		chart, err = helmClient.GetChart(context.TODO(), k8sClient, server.URL, "", "", "argocd")
		Expect(chart).Should(BeNil())
		Expect(err).ShouldNot(BeNil())

		chart, err = helmClient.GetChart(
			context.TODO(),
			k8sClient,
			server.URL,
			"hypershift-template",
			"0.0.2",
			"argocd",
		)
		Expect(chart).ShouldNot(BeNil())
		Expect(err).Should(BeNil())
	})
	It("GetChart with repo secret", func() {
		helmClient := CreateHelmClient(k8sManager, cfg)
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "argocd",
				Labels: map[string]string{
					argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
				},
			},
			Data: map[string][]byte{
				"type": []byte("helm"),
				"url":  []byte(server.URL),
			},
		}

		client := fake.NewFakeClientWithScheme(scheme.Scheme, secret)
		chart, err := helmClient.GetChart(
			context.TODO(),
			client,
			server.URL,
			"hypershift-template",
			"0.0.2",
			"argocd",
		)
		Expect(chart).ShouldNot(BeNil())
		Expect(err).Should(BeNil())
	})
	It("GetChart https with repo secret - insecure", func() {
		helmClient := CreateHelmClient(k8sManager, cfg)
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "argocd",
				Labels: map[string]string{
					argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
				},
			},
			Data: map[string][]byte{
				"type":     []byte("helm"),
				"url":      []byte(httpsServer.URL),
				"insecure": []byte("true"),
			},
		}

		client := fake.NewFakeClientWithScheme(scheme.Scheme, secret)
		chart, err := helmClient.GetChart(
			context.TODO(),
			client,
			httpsServer.URL,
			"hypershift-template",
			"0.0.2",
			"argocd",
		)
		Expect(chart).ShouldNot(BeNil())
		Expect(err).Should(BeNil())
	})
	It("GetChart https with repo secret and ca cert", func() {
		helmClient := CreateHelmClient(k8sManager, cfg)
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "argocd",
				Labels: map[string]string{
					argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
				},
			},
			Data: map[string][]byte{
				"type": []byte("helm"),
				"url":  []byte(httpsServer.URL),
			},
		}
		parsedUrl, err := url.ParseRequestURI(httpsServer.URL)
		if err != nil {
			Fail(err.Error())
		}

		data, err := os.ReadFile("../testutils/helm/server.crt")
		if err != nil {
			Fail(err.Error())
		}
		cm := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      "argocd-tls-certs-cm",
				Namespace: "argocd",
			},
			Data: map[string]string{
				parsedUrl.Host: string(data),
			},
		}

		client := fake.NewFakeClientWithScheme(scheme.Scheme, secret, cm)

		chart, err := helmClient.GetChart(
			context.TODO(),
			client,
			httpsServer.URL,
			"hypershift-template",
			"0.0.2",
			"argocd",
		)
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

	return NewHelmClient(
		config,
		k8sManager.GetClient(),
		&certDataFileName,
		&keyDataFileName,
		&caDataFileName,
	)
}
