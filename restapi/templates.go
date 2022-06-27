package restapi

import (
	gin "github.com/gin-gonic/gin"
	"github.com/rawagner/cluster-templates-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TemplatesResponse struct {
	Type      string
	Available int32
}

// cr metadata
func (h *K8sHandler) getClusterTemplates(c *gin.Context) {
	username := c.GetString("username")

	template := &v1alpha1.ClusterTemplateQuota{}

	err := h.ControllerClient.Get(c, client.ObjectKey{Name: username, Namespace: "default"}, template)
	if err != nil {
		c.AbortWithError(404, err)
	}

	response := []TemplatesResponse{}
	/*

		for key, element := range template.Status.Quota {
			response = append(response, TemplatesResponse{Type: key, Available: template.Spec.Quota[key].Count - element.Count})
		}
	*/

	c.JSON(200, &response)
}
