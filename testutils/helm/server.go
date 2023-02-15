package helm

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
)

func handlerFunc(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/index.yaml" {
		if u, p, ok := r.BasicAuth(); ok {
			if !(u == "admin" && p == "password") {
				fmt.Println("Invalid username/password")
				w.WriteHeader(401)
				return
			}
		}

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

/*
File testutils/git/base contains data which would be returned by server.
If client perform command: git ls-remote http://githubserver/machacekondra/myapp.
First client send request: GET /machacekondra/myapp/info/refs
The server should respond with the binary data contained in the file testutils/git/base
and the header - Content-Type: application/x-git-upload-pack-advertisement
Then client send POST /machacekondra/myapp/info/refs
server should send empty response with header - Content-Type: application/x-git-upload-pack-request

More info: https://git-scm.com/docs/http-protocol/
*/
func gitHandlerFunc(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/machacekondra/myapp/info/refs") {
		// Token auth:
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			fmt.Println(authHeader[len("Bearer "):])
			if authHeader[len("Bearer "):] != "tokenXYZ" {
				w.WriteHeader(401)
				return
			}
		}
		// Basic auth:
		if u, p, ok := r.BasicAuth(); ok {
			if !(u == "admin" && p == "password") {
				fmt.Println("Invalid username/password")
				w.WriteHeader(401)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/x-git-upload-pack-request")
		} else if r.Method == "GET" {
			path, err := filepath.Abs("../testutils/git/base")
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
			w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
			w.Write(indexData)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Nothing to show"))
	}
}

func StartGitRepoServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(gitHandlerFunc))
	return server
}

func StartTLSGitRepoServer() *httptest.Server {
	server := httptest.NewUnstartedServer(http.HandlerFunc(gitHandlerFunc))
	cert, err := tls.LoadX509KeyPair("../testutils/helm/server.crt", "../testutils/helm/server.key")
	if err != nil {
		Fail(err.Error())
	}
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	server.StartTLS()
	return server
}

func StartHelmRepoServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(handlerFunc))
	return server
}

func StartTLSHelmRepoServer() *httptest.Server {
	server := httptest.NewUnstartedServer(http.HandlerFunc(handlerFunc))
	cert, err := tls.LoadX509KeyPair("../testutils/helm/server.crt", "../testutils/helm/server.key")
	if err != nil {
		Fail(err.Error())
	}
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	server.StartTLS()
	return server
}

func StartMTLSHelmRepoServer() *httptest.Server {
	server := httptest.NewUnstartedServer(http.HandlerFunc(handlerFunc))
	serverCert, err := tls.LoadX509KeyPair(
		"../testutils/helm/server.crt",
		"../testutils/helm/server.key",
	)
	if err != nil {
		Fail(err.Error())
	}
	caCert, err := ioutil.ReadFile("../testutils/helm/ca.crt")
	if err != nil {
		Fail(err.Error())
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
	}
	server.StartTLS()
	return server
}
