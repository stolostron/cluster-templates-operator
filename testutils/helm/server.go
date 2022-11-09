package helm

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
)

func StartHelmRepoServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.yaml" {
			path, err := filepath.Abs("../testutils/helm/index.yaml")
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			indexData, err := os.ReadFile(path)
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write(indexData)
		} else {
			data, err := os.ReadFile("../testutils/helm/" + r.URL.Path)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		}
	}))
	return server
}
