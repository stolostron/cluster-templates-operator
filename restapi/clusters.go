package restapi

import (
	gin "github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rawagner/cluster-templates-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterReponse struct {
	Name              string
	Type              string
	KubeadminPassword string
	URL               string
	Status            string
}

func (h *K8sHandler) getClusters(c *gin.Context) {
	username := c.GetString("username")

	instanceList := &v1alpha1.ClusterTemplateInstanceList{}
	opts := []client.ListOption{
		client.MatchingLabels{"username": username},
	}

	err := h.ControllerClient.List(c, instanceList, opts...)

	if err != nil {
		c.AbortWithError(500, err)
	}

	var response []ClusterReponse

	for _, instance := range instanceList.Items {
		response = append(response, ClusterReponse{
			Name:              instance.Name,
			Type:              instance.Labels["type"],
			URL:               instance.Status.APIserverURL,
			KubeadminPassword: instance.Status.KubeadminPassword,
			Status:            instance.Status.ClusterStatus,
		})
	}

	c.JSON(200, &response)
}

type CreateClusterBody struct {
	Type        string                 `json:"type"`
	ClusterName string                 `json:"clusterName"`
	Values      map[string]interface{} `json:"values"`
}

func (h *K8sHandler) createCluster(c *gin.Context) {
	createClusterBody := &CreateClusterBody{}
	c.ShouldBindBodyWith(createClusterBody, binding.JSON)

	username := c.GetString("username")

	template := &v1alpha1.ClusterTemplateQuota{}

	err := h.ControllerClient.Get(c, client.ObjectKey{Name: username, Namespace: "default"}, template)

	if err != nil {
		c.AbortWithError(500, err)
	}

	// templateURL := template.Spec.Quota[createClusterBody.Type].HelmRepositoryRef

	labels := make(map[string]string)
	labels["username"] = username
	labels["type"] = createClusterBody.Type
	// values, err := json.Marshal(createClusterBody.Values)

	if err != nil {
		c.AbortWithError(500, err)
	}

	instance := &v1alpha1.ClusterTemplateInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      createClusterBody.ClusterName,
			Namespace: "default",
			Labels:    labels,
		},
		Spec: v1alpha1.ClusterTemplateInstanceSpec{
			// HelmRepositoryRef: templateURL,
			// Values:            string(values),
		},
	}

	err = h.ControllerClient.Create(c, instance)

	if err != nil {
		c.String(403, err.Error())
	}
	c.String(202, "Created")
}

type DeleteClusterBody struct {
	ClusterName string `json:"clusterName"`
}

func (h *K8sHandler) deleteCluster(c *gin.Context) {
	deleteClusterBody := &DeleteClusterBody{}
	c.ShouldBindBodyWith(deleteClusterBody, binding.JSON)

	username := c.GetString("username")

	clusterInstance := &v1alpha1.ClusterTemplateInstance{}

	err := h.ControllerClient.Get(c, client.ObjectKey{Name: deleteClusterBody.ClusterName, Namespace: "default"}, clusterInstance)

	if err != nil {
		c.AbortWithError(500, err)
	}

	if clusterInstance.Labels["username"] != username {
		c.AbortWithStatus(404)
	}

	err = h.ControllerClient.Delete(c, clusterInstance)
	if err != nil {
		c.AbortWithError(500, err)
	}
}

func (h *K8sHandler) testHandler(c *gin.Context) {
	c.String(200, "OK!")
}
