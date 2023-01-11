package bridge

import (
	"net/http"
	"os"

	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

var bridgeLog = ctrl.Log.WithName("bridge")

func RunServer(config *rest.Config, tlsCertFile string, tlsKeyFile string) *http.Server {
	router := GetRouter(config)
	server := &http.Server{Addr: ":8001", Handler: router}
	go func(server *http.Server, tlsCertFile string, tlsKeyFile string) {
		var err error
		if tlsCertFile != "" && tlsKeyFile != "" {
			err = server.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil {
			bridgeLog.Error(err, "Listen and serve failed")
			os.Exit(1)
		}
	}(server, tlsCertFile, tlsKeyFile)
	return server
}
