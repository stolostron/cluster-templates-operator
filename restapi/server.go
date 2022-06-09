package restapi

import (
	"fmt"

	gin "github.com/gin-gonic/gin"
	"github.com/rawagner/cluster-templates-operator/helm"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type K8sHandler struct {
	K8sClient *kubernetes.Clientset
	// DynamicK8sClient dynamic.Interface
	ControllerClient client.Client
	helmClient       *helm.HelmClient
}

func StartHTTPServer(config *rest.Config, scheme *runtime.Scheme, helmClient *helm.HelmClient) {
	handler := newHandler(config, scheme, helmClient)

	router := gin.New()
	//middleware - check user token
	router.Use(handler.validateUserToken)

	router.POST("/templates", handler.getClusterTemplates)
	router.POST("/clusters", handler.getClusters)
	router.POST("/create-cluster", handler.createCluster)
	router.POST("/delete-cluster", handler.deleteCluster)
	router.GET("/test", handler.testHandler)

	router.Run(":8090")
}

func newHandler(config *rest.Config, scheme *runtime.Scheme, helmClient *helm.HelmClient) *K8sHandler {
	// create the clientset
	// dynamicClient, err := dynamic.NewForConfig(config)
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	controllerClient, err := client.New(config, client.Options{
		Scheme: scheme,
	})

	if err != nil {
		fmt.Println(err)
	}

	return &K8sHandler{
		K8sClient: clientset,
		// DynamicK8sClient: dynamicClient,
		ControllerClient: controllerClient,
		helmClient:       helmClient,
	}

}
