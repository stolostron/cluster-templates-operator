package helmserver

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
			path, err := filepath.Abs("../testutils/helmserver/index.yaml")
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
		}
		if r.URL.Path == "/hypershift-template-0.1.0.tgz" {
			data, err := os.ReadFile("../testutils/helmserver/hypershift-template-0.1.0.tgz")
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
