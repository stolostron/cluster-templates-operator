package repository

import (
	"net/http/httptest"
	"net/url"
	"os"

	argoCommon "github.com/argoproj/argo-cd/v2/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	helmserver "github.com/stolostron/cluster-templates-operator/testutils/helm"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Git client", func() {
	var server *httptest.Server
	var httpsServer *httptest.Server
	var err error
	var httpClient *HttpClient
	var httpsClient *HttpClient
	BeforeEach(func() {
		server = helmserver.StartGitRepoServer()
		httpsServer = helmserver.StartTLSGitRepoServer()
		httpClient, err = GetRepoHTTPClient(server.URL, nil, nil)
		Expect(err).To(BeNil())
		httpsClient, err = GetRepoHTTPClient(httpsServer.URL, nil, nil)
		Expect(err).To(BeNil())
	})
	AfterEach(func() {
		server.Close()
		httpsServer.Close()
	})
	It("GetGitInfo", func() {
		// RepoURL = ""
		tags, branches, err := GetGitInfo(httpClient, "")
		Expect(tags).Should(BeNil())
		Expect(branches).Should(BeNil())
		Expect(err).ShouldNot(BeNil())

		// RepoURL = server.URL
		repoURL := server.URL + "/machacekondra/myapp"
		tags, branches, err = GetGitInfo(httpClient, repoURL)
		Expect(err).Should(BeNil())
		Expect(tags).ShouldNot(BeNil())
		Expect(tags).Should(Equal([]string{"1.0.0"}))
		Expect(branches).ShouldNot(BeNil())
		Expect(branches).Should(ContainElements([]string{"main", "test"}))
	})
	It("GetGitInfo with repo secret - username/password", func() {
		repoURL := server.URL + "/machacekondra/myapp"
		httpClient.secret = &corev1.Secret{
			ObjectMeta: getMeta(),
			Data: map[string][]byte{
				"type":     []byte("git"),
				"url":      []byte(repoURL),
				"username": []byte("admin"),
				"password": []byte("password"),
			},
		}

		tags, branches, err := GetGitInfo(httpClient, repoURL)
		Expect(err).Should(BeNil())
		Expect(tags).ShouldNot(BeNil())
		Expect(tags).Should(Equal([]string{"1.0.0"}))
		Expect(branches).ShouldNot(BeNil())
		Expect(branches).Should(ContainElements([]string{"main", "test"}))
	})
	It("GetGitInfo with repo secret - token", func() {
		repoURL := server.URL + "/machacekondra/myapp"
		httpClient.secret = &corev1.Secret{
			ObjectMeta: getMeta(),
			Data: map[string][]byte{
				"type":     []byte("git"),
				"url":      []byte(repoURL),
				"username": []byte(""),
				"password": []byte("tokenXYZ"),
			},
		}

		tags, branches, err := GetGitInfo(httpClient, repoURL)
		Expect(err).Should(BeNil())
		Expect(tags).ShouldNot(BeNil())
		Expect(tags).Should(Equal([]string{"1.0.0"}))
		Expect(branches).ShouldNot(BeNil())
		Expect(branches).Should(ContainElements([]string{"main", "test"}))
	})
	It("GetGitInfo with repo secret - invalid username/password", func() {
		repoURL := server.URL + "/machacekondra/myapp"
		httpClient.secret = &corev1.Secret{
			ObjectMeta: getMeta(),
			Data: map[string][]byte{
				"type":     []byte("git"),
				"url":      []byte(repoURL),
				"username": []byte("admin"),
				"password": []byte("incorrect"),
			},
		}

		tags, branches, err := GetGitInfo(httpClient, repoURL)
		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(ContainSubstring("authentication required"))
		Expect(tags).Should(BeNil())
		Expect(branches).Should(BeNil())
	})
	It("GetGitInfo with repo secret - invalid token", func() {
		repoURL := server.URL + "/machacekondra/myapp"
		httpClient.secret = &corev1.Secret{
			ObjectMeta: getMeta(),
			Data: map[string][]byte{
				"type":     []byte("git"),
				"url":      []byte(repoURL),
				"username": []byte(""),
				"password": []byte("incorrect"),
			},
		}

		tags, branches, err := GetGitInfo(httpClient, repoURL)
		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(ContainSubstring("authentication required"))
		Expect(tags).Should(BeNil())
		Expect(branches).Should(BeNil())
	})
	It("GetGitInfo https with repo secret - insecure", func() {
		repoURL := server.URL + "/machacekondra/myapp"
		httpsClient.secret = &corev1.Secret{
			ObjectMeta: getMeta(),
			Data: map[string][]byte{
				"type":     []byte("git"),
				"url":      []byte(repoURL),
				"insecure": []byte("true"),
			},
		}

		tags, branches, err := GetGitInfo(httpsClient, repoURL)
		Expect(err).Should(BeNil())
		Expect(tags).ShouldNot(BeNil())
		Expect(tags).Should(Equal([]string{"1.0.0"}))
		Expect(branches).ShouldNot(BeNil())
		Expect(branches).Should(ContainElements([]string{"main", "test"}))
	})
	It("GetGitInfo https with repo secret and ca cert", func() {
		repoURL := httpsServer.URL + "/machacekondra/myapp"
		parsedUrl, err := url.ParseRequestURI(repoURL)
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
		httpsClient, err = GetRepoHTTPClient(repoURL, nil, cm)
		if err != nil {
			Fail(err.Error())
		}
		httpsClient.secret = &corev1.Secret{
			ObjectMeta: getMeta(),
			Data: map[string][]byte{
				"type": []byte("helm"),
				"url":  []byte(httpsServer.URL),
			},
		}

		tags, branches, err := GetGitInfo(httpsClient, repoURL)
		Expect(err).Should(BeNil())
		Expect(tags).ShouldNot(BeNil())
		Expect(tags).Should(Equal([]string{"1.0.0"}))
		Expect(branches).ShouldNot(BeNil())
		Expect(branches).Should(ContainElements([]string{"main", "test"}))
	})
})

func getMeta() v1.ObjectMeta {
	return v1.ObjectMeta{
		Name:      "foo",
		Namespace: "argocd",
		Labels: map[string]string{
			argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
		},
	}
}
