package restapi

import (
	"fmt"

	gin "github.com/gin-gonic/gin"
	"github.com/rawagner/cluster-templates-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/* helm metadata
func (h *K8sHandler) getClusterTemplates(c *gin.Context) {
	fmt.Println("handler1")
	username := c.GetString("username")

	fmt.Println(username)
	template := &v1alpha1.ClusterTemplateQuota{}

	err := h.ControllerClient.Get(c, client.ObjectKey{Name: username, Namespace: "default"}, template)
	fmt.Println("handler1-1")
	if err != nil {
		fmt.Println("handler1-2")
		c.AbortWithError(404, err)
	}

	chart, err := h.helmClient.GetChart(template.Spec.Quota["foo"].HelmRepositoryRef)
	fmt.Println("handler1-3")
	if err != nil {
		fmt.Println("helm err1")
		fmt.Println(err)
	}

	fmt.Println(chart.Metadata.Name)

	c.JSON(200, &chart.Metadata)
}
*/

type TemplatesResponse struct {
	Type      string
	Available int32
}

// cr metadata
func (h *K8sHandler) getClusterTemplates(c *gin.Context) {
	fmt.Println("handler1")
	username := c.GetString("username")

	fmt.Println(username)
	template := &v1alpha1.ClusterTemplateQuota{}

	err := h.ControllerClient.Get(c, client.ObjectKey{Name: username, Namespace: "default"}, template)
	fmt.Println("handler1-1")
	if err != nil {
		fmt.Println("handler1-2")
		c.AbortWithError(404, err)
	}

	fmt.Println(template.Spec)

	response := []TemplatesResponse{}

	for key, element := range template.Status.Quota {
		response = append(response, TemplatesResponse{Type: key, Available: template.Spec.Quota[key].Count - element.Count})
	}

	c.JSON(200, &response)
}
