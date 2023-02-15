package repository

import (
	"context"
	"net/http/httptest"
	"net/url"

	argoCommon "github.com/argoproj/argo-cd/v2/common"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	helmserver "github.com/stolostron/cluster-templates-operator/testutils/helm"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Helm client", func() {
	var server *httptest.Server
	var httpsServer *httptest.Server
	BeforeEach(func() {
		server = helmserver.StartHelmRepoServer()
		httpsServer = helmserver.StartTLSHelmRepoServer()
	})
	AfterEach(func() {
		server.Close()
		httpsServer.Close()
	})
	It("GetChart", func() {
		chart, err := GetChart(context.TODO(), k8sClient, "", "", "", "argocd")
		Expect(chart).Should(BeNil())
		Expect(err).ShouldNot(BeNil())
		server := helmserver.StartHelmRepoServer()

		chart, err = GetChart(context.TODO(), k8sClient, server.URL, "", "", "argocd")
		Expect(chart).Should(BeNil())
		Expect(err).ShouldNot(BeNil())

		chart, err = GetChart(
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
		chart, err := GetChart(
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
		chart, err := GetChart(
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

		data, err := os.ReadFile("../testutils/helm/ca.crt")
		if err != nil {
			Fail(err.Error())
		}
		cm := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      RepoCMName,
				Namespace: "argocd",
			},
			Data: map[string]string{
				parsedUrl.Hostname(): string(data),
			},
		}

		client := fake.NewFakeClientWithScheme(scheme.Scheme, secret, cm)

		chart, err := GetChart(
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
	It("GetChart protected by basic auth", func() {
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
				"url":      []byte(server.URL),
				"username": []byte("admin"),
				"password": []byte("password"),
			},
		}
		client := fake.NewFakeClientWithScheme(scheme.Scheme, secret)

		chart, err := GetChart(
			context.TODO(),
			client,
			server.URL,
			"hypershift-template",
			"0.0.2",
			"argocd",
		)
		Expect(err).Should(BeNil())
		Expect(chart).ShouldNot(BeNil())
	})
	It("GetChart protected by basic auth invalid username", func() {
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
				"url":      []byte(server.URL),
				"username": []byte("admin"),
				"password": []byte("invalid"),
			},
		}
		client := fake.NewFakeClientWithScheme(scheme.Scheme, secret)

		chart, err := GetChart(
			context.TODO(),
			client,
			server.URL,
			"hypershift-template",
			"0.0.2",
			"argocd",
		)
		Expect(chart).Should(BeNil())
		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(ContainSubstring("401 Unauthorized"))
	})
})
