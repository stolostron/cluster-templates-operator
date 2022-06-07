package restapi

import (
	"fmt"

	gin "github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TokenBody struct {
	Token string `json:"token"`
}

func (h *K8sHandler) validateUserToken(c *gin.Context) {
	userToken := &TokenBody{}
	c.ShouldBindBodyWith(userToken, binding.JSON)

	tokenReview := &authv1.TokenReview{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-deployment",
		},
		Spec: authv1.TokenReviewSpec{
			Token: userToken.Token,
		},
	}

	tokenResult, err := h.K8sClient.AuthenticationV1().TokenReviews().Create(c, tokenReview, metav1.CreateOptions{})
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(c.Writer, "Error creating token review")
		c.Abort()
		return

	}

	if !tokenResult.Status.Authenticated {
		fmt.Fprint(c.Writer, "Token not valid")
		c.Abort()
		return
	}

	c.Set("username", tokenResult.Status.User.Username)
	c.Next()
}
