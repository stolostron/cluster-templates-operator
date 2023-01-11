package bridge

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	argoCommon "github.com/argoproj/argo-cd/v2/common"
	testutils "github.com/stolostron/cluster-templates-operator/testutils"
	helm "github.com/stolostron/cluster-templates-operator/testutils/helm"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func fetchRepositories(client *http.Client) []RepositoryIndex {
	req, err := http.NewRequest("GET", server.URL+repositoriesAPI, nil)
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Authorization", "Bearer foo")
	resp, err := client.Do(req)
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	repositories := []RepositoryIndex{}
	err = yaml.Unmarshal(body, &repositories)
	Expect(err).ToNot(HaveOccurred())
	return repositories
}

func fetchRepository(client *http.Client, secretName string) RepositoryIndex {
	req, err := http.NewRequest("GET", server.URL+repositoryAPI+"/"+secretName, nil)
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Authorization", "Bearer foo")
	resp, err := client.Do(req)
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	repository := RepositoryIndex{}
	err = yaml.Unmarshal(body, &repository)
	Expect(err).ToNot(HaveOccurred())
	return repository
}

func createResource(obj k8s.Object) {
	err := k8sClient.Create(ctx, obj)
	Expect(err).ToNot(HaveOccurred())
	resourcesToDelete = append(resourcesToDelete, obj)
}

var resourcesToDelete []k8s.Object

var _ = Describe("Repo bridge", func() {
	var client *http.Client
	BeforeEach(func() {
		client = &http.Client{}
	})
	AfterEach(func() {
		for _, res := range resourcesToDelete {
			testutils.DeleteResource(ctx, res, k8sClient)
		}
		resourcesToDelete = []k8s.Object{}
	})
	It("Rejects when Auth header is missing", func() {
		resp, err := http.Get(server.URL + repositoriesAPI)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		resp, err = http.Get(server.URL + repositoryAPI + "/foo")
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	})
	It("Rejects when Auth header is incorrect", func() {
		req, err := http.NewRequest("GET", server.URL+repositoriesAPI, nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "foo")
		resp, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "argocd",
				Labels: map[string]string{
					argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
				},
			},
			Data: map[string][]byte{},
		}
		createResource(secret)
		resp, err = http.Get(server.URL + repositoryAPI + "/" + secret.Name)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	})
	It("Empty repositories", func() {
		repositories := fetchRepositories(client)
		Expect(repositories).To(BeEmpty())
	})
	It("Repository does not exist", func() {
		req, err := http.NewRequest("GET", server.URL+repositoryAPI+"/foo", nil)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Authorization", "Bearer foo")
		resp, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	})
	It("Repository secret with invalid data", func() {
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "argocd",
				Labels: map[string]string{
					argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
				},
			},
			Data: map[string][]byte{},
		}
		createResource(secret)
		repositories := fetchRepositories(client)
		Expect(len(repositories)).To(Equal(1))

		verifyRepo := func(repository RepositoryIndex) {
			Expect(repository.Name).To(Equal(""))
			Expect(repository.Url).To(Equal(""))
			Expect(repository.Index).To(BeNil())
			Expect(repository.Error).NotTo(Equal(""))
		}
		verifyRepo(repositories[0])

		repository := fetchRepository(client, secret.Name)
		verifyRepo(repository)
	})
	It("Repository secret repo", func() {
		server := helm.StartHelmRepoServer()
		defer server.Close()
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "public-repo",
				Namespace: "argocd",
				Labels: map[string]string{
					argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
				},
			},
			Data: map[string][]byte{
				"name": []byte("foo"),
				"url":  []byte(server.URL),
			},
		}
		createResource(secret)
		repositories := fetchRepositories(client)
		Expect(len(repositories)).To(Equal(1))

		verifyRepo := func(repository RepositoryIndex) {
			Expect(repository.Name).To(Equal("foo"))
			Expect(repository.Url).To(Equal(server.URL))
			Expect(repository.Index).ToNot(BeNil())
			Expect(repository.Error).To(Equal(""))
		}
		verifyRepo(repositories[0])

		repository := fetchRepository(client, secret.Name)
		verifyRepo(repository)
	})
	It("Repository secret with tls repo - insecure conn - allowed", func() {
		server := helm.StartTLSHelmRepoServer()
		defer server.Close()
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "public-repo",
				Namespace: "argocd",
				Labels: map[string]string{
					argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
				},
			},
			Data: map[string][]byte{
				"name":     []byte("foo"),
				"url":      []byte(server.URL),
				"insecure": []byte("true"),
			},
		}
		createResource(secret)
		repositories := fetchRepositories(client)
		Expect(len(repositories)).To(Equal(1))

		verifyRepo := func(repository RepositoryIndex) {
			Expect(repository.Name).To(Equal("foo"))
			Expect(repository.Url).To(Equal(server.URL))
			Expect(repository.Index).ToNot(BeNil())
			Expect(repository.Error).To(Equal(""))
		}
		verifyRepo(repositories[0])

		repository := fetchRepository(client, secret.Name)
		verifyRepo(repository)
	})
	It("Repository secret with tls repo - insecure conn - disallowed", func() {
		server := helm.StartTLSHelmRepoServer()
		defer server.Close()
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "public-repo",
				Namespace: "argocd",
				Labels: map[string]string{
					argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
				},
			},
			Data: map[string][]byte{
				"name": []byte("foo"),
				"url":  []byte(server.URL),
			},
		}
		createResource(secret)
		repositories := fetchRepositories(client)
		Expect(len(repositories)).To(Equal(1))

		verifyRepo := func(repository RepositoryIndex) {
			Expect(repository.Name).To(Equal("foo"))
			Expect(repository.Url).To(Equal(server.URL))
			Expect(repository.Index).To(BeNil())
			Expect(repository.Error).NotTo(Equal(""))
		}
		verifyRepo(repositories[0])

		repository := fetchRepository(client, secret.Name)
		verifyRepo(repository)
	})
	It("Repository secret with tls repo - with cm", func() {
		server := helm.StartTLSHelmRepoServer()
		defer server.Close()
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "public-repo",
				Namespace: "argocd",
				Labels: map[string]string{
					argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
				},
			},
			Data: map[string][]byte{
				"name": []byte("foo"),
				"url":  []byte(server.URL),
			},
		}
		createResource(secret)
		parsedUrl, err := url.ParseRequestURI(server.URL)
		Expect(err).ToNot(HaveOccurred())

		file, err := ioutil.ReadFile("../testutils/helm/ca.crt")
		Expect(err).ToNot(HaveOccurred())

		cm := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      "argocd-tls-certs-cm",
				Namespace: "argocd",
			},
			Data: map[string]string{
				parsedUrl.Hostname(): string(file),
			},
		}
		createResource(cm)
		repositories := fetchRepositories(client)
		Expect(len(repositories)).To(Equal(1))

		verifyRepo := func(repository RepositoryIndex) {
			Expect(repository.Name).To(Equal("foo"))
			Expect(repository.Url).To(Equal(server.URL))
			Expect(repository.Index).NotTo(BeNil())
			Expect(repository.Error).To(Equal(""))
		}
		verifyRepo(repositories[0])

		repository := fetchRepository(client, secret.Name)
		verifyRepo(repository)
	})
	It("Repository secret with tls repo - with cm - invalid content", func() {
		server := helm.StartTLSHelmRepoServer()
		defer server.Close()
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "public-repo",
				Namespace: "argocd",
				Labels: map[string]string{
					argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
				},
			},
			Data: map[string][]byte{
				"name": []byte("foo"),
				"url":  []byte(server.URL),
			},
		}
		createResource(secret)
		parsedUrl, err := url.ParseRequestURI(server.URL)
		Expect(err).ToNot(HaveOccurred())

		cm := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      "argocd-tls-certs-cm",
				Namespace: "argocd",
			},
			Data: map[string]string{
				parsedUrl.Hostname(): "invalid_content",
			},
		}
		createResource(cm)
		repositories := fetchRepositories(client)
		Expect(len(repositories)).To(Equal(1))

		verifyRepo := func(repository RepositoryIndex) {
			Expect(repository.Name).To(Equal("foo"))
			Expect(repository.Url).To(Equal(server.URL))
			Expect(repository.Index).To(BeNil())
			Expect(repository.Error).NotTo(Equal(""))
		}
		verifyRepo(repositories[0])

		repository := fetchRepository(client, secret.Name)
		verifyRepo(repository)
	})
	It("Repository secret with mtls repo", func() {
		server := helm.StartTLSHelmRepoServer()
		defer server.Close()

		crtFile, err := ioutil.ReadFile("../testutils/helm/client.crt")
		Expect(err).ToNot(HaveOccurred())
		keyFile, err := ioutil.ReadFile("../testutils/helm/client.key")
		Expect(err).ToNot(HaveOccurred())

		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "private-repo",
				Namespace: "argocd",
				Labels: map[string]string{
					argoCommon.LabelKeySecretType: argoCommon.LabelValueSecretTypeRepository,
				},
			},
			Data: map[string][]byte{
				"name":              []byte("foo"),
				"url":               []byte(server.URL),
				"tlsClientCertKey":  keyFile,
				"tlsClientCertData": crtFile,
			},
		}
		createResource(secret)
		parsedUrl, err := url.ParseRequestURI(server.URL)
		Expect(err).ToNot(HaveOccurred())

		file, err := ioutil.ReadFile("../testutils/helm/ca.crt")
		Expect(err).ToNot(HaveOccurred())

		cm := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      "argocd-tls-certs-cm",
				Namespace: "argocd",
			},
			Data: map[string]string{
				parsedUrl.Hostname(): string(file),
			},
		}
		createResource(cm)
		repositories := fetchRepositories(client)
		Expect(len(repositories)).To(Equal(1))

		verifyRepo := func(repository RepositoryIndex) {
			Expect(repository.Name).To(Equal("foo"))
			Expect(repository.Url).To(Equal(server.URL))
			Expect(repository.Index).NotTo(BeNil())
			Expect(repository.Error).To(Equal(""))
		}
		verifyRepo(repositories[0])

		repository := fetchRepository(client, secret.Name)
		verifyRepo(repository)
	})
	// test basic auth

})
