package helm

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
)

func handlerFunc(w http.ResponseWriter, r *http.Request) {
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
}

func StartHelmRepoServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(handlerFunc))
	return server
}

func StartHttpsHelmRepoServer() *httptest.Server {
	server := httptest.NewUnstartedServer(http.HandlerFunc(handlerFunc))
	cert, err := tls.LoadX509KeyPair("../testutils/helm/server.crt", "../testutils/helm/server.key")
	if err != nil {
		Fail(err.Error())
	}
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	server.StartTLS()
	return server
}
