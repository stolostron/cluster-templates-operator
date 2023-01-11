package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	argoCommon "github.com/argoproj/argo-cd/v2/common"
	"github.com/julienschmidt/httprouter"
	controllers "github.com/stolostron/cluster-templates-operator/controllers"
	helm "github.com/stolostron/cluster-templates-operator/helm"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

const (
	maxConcurrent   = 15
	repositoriesAPI = "/api/helm-repositories"
	repositoryAPI   = "/api/helm-repository"
)

type RepositoryIndex struct {
	Index *repo.IndexFile `json:"index,omitempty"`
	Error string          `json:"error,omitempty"`
	Url   string          `json:"url,omitempty"`
	Name  string          `json:"name,omitempty"`
}

func writeError(w http.ResponseWriter, errorMsg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	enc, err := json.Marshal(errorMsg)
	if err != nil {
		bridgeLog.Info(fmt.Sprintf("Failed JSON-encoding HTTP response: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(code)
		w.Write(enc)
	}
}

func getRepositoryIndex(
	ctx context.Context,
	secret *corev1.Secret,
	cm *corev1.ConfigMap,
) RepositoryIndex {
	repoName := string(secret.Data["name"])
	repoURL := string(secret.Data["url"])
	repository := RepositoryIndex{
		Url:  repoURL,
		Name: repoName,
	}
	httpClient, err := helm.GetRepoHTTPClient(
		ctx,
		repoURL,
		[]corev1.Secret{*secret},
		cm,
	)
	if err != nil {
		repository.Error = err.Error()
	} else {
		indexFile, err := helm.GetIndexFile(httpClient, repoURL)
		if err != nil {
			repository.Error = err.Error()
		}
		repository.Index = indexFile
	}
	return repository
}

func getRepo(
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
	k8sClient *kubernetes.Clientset,
) {
	secretName := params.ByName("name")

	secret, err := k8sClient.CoreV1().
		Secrets(controllers.ArgoCDNamespace).
		Get(r.Context(), secretName, metav1.GetOptions{})

	if err != nil {
		code := http.StatusInternalServerError
		if apierrors.IsNotFound(err) {
			code = http.StatusNotFound
		}
		if apierrors.IsForbidden(err) {
			code = http.StatusForbidden
		}
		writeError(w, "Failed to get repository secret: "+err.Error(), code)
		return
	}

	cm, err := k8sClient.CoreV1().
		ConfigMaps(controllers.ArgoCDNamespace).
		Get(r.Context(), helm.RepoCMName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		writeError(
			w,
			"Failed to get repositories config map: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}
	repository := getRepositoryIndex(r.Context(), secret, cm)
	out, err := yaml.Marshal(repository)
	if err != nil {
		writeError(w, "Failed to deserialize index file to yaml", http.StatusInternalServerError)
		return
	}

	w.Write(out)
}

func getRepositories(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
	k8sClient *kubernetes.Clientset,
) {
	ctx := r.Context()
	secrets, err := k8sClient.CoreV1().
		Secrets(controllers.ArgoCDNamespace).
		List(ctx, metav1.ListOptions{LabelSelector: argoCommon.LabelKeySecretType + "=" + argoCommon.LabelValueSecretTypeRepository})
	if err != nil {
		writeError(
			w,
			"Failed to get repositories info: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	cm, err := k8sClient.CoreV1().
		ConfigMaps(controllers.ArgoCDNamespace).
		Get(r.Context(), helm.RepoCMName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		writeError(
			w,
			"Failed to get repositories config map: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}
	repositories := make([]RepositoryIndex, len(secrets.Items))

	guard := make(chan struct{}, maxConcurrent)
	wg := sync.WaitGroup{}
	for index, secret := range secrets.Items {
		guard <- struct{}{}
		wg.Add(1)
		go func(index int, secret corev1.Secret) {
			defer wg.Done()
			repositories[index] = getRepositoryIndex(r.Context(), &secret, cm)
			<-guard
		}(index, secret)
	}
	wg.Wait()

	out, err := yaml.Marshal(repositories)
	if err != nil {
		writeError(w, "Failed to deserialize index file to yaml", http.StatusInternalServerError)
		return
	}

	w.Write(out)
}

type HandleWithToken func(http.ResponseWriter, *http.Request, httprouter.Params, *kubernetes.Clientset)

func withUserClient(h HandleWithToken, config rest.Config) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, "Missing user's bearer token header", http.StatusUnauthorized)
			return
		}
		bearerToken := strings.Split(authHeader, "Bearer ")
		if len(bearerToken) != 2 || bearerToken[1] == "" {
			writeError(w, "Missing user's bearer token header", http.StatusUnauthorized)
			return
		}
		client, err := CreateTypedClient(bearerToken[1], config)
		if err != nil {
			writeError(
				w,
				"Failed to create user k8s client: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
		h(w, r, params, client)
	}
}

func GetRouter(config *rest.Config) *httprouter.Router {
	router := httprouter.New()
	router.GET(repositoryAPI+"/:name", withUserClient(getRepo, *config))
	router.GET(repositoriesAPI, withUserClient(getRepositories, *config))
	return router
}
