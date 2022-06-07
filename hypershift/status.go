package hypershift

import (
	"fmt"

	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

type KubeSecret struct {
	Namespace    string
	PassSecret   string
	ConfigSecret string
}

func GetHypershiftInfo(manifest string) (*KubeSecret, error) {
	var hostedC hypershiftv1alpha1.HostedCluster
	err := k8sYaml.Unmarshal([]byte(manifest), &hostedC)
	if err != nil {
		fmt.Println("ERR decoding hostedC")
		return nil, err
	}
	fmt.Println("+++++++FOUND+++++")

	return &KubeSecret{
		Namespace:    hostedC.Namespace + "-" + hostedC.Name,
		PassSecret:   "kubeadmin-password",
		ConfigSecret: "admin-kubeconfig",
	}, nil
}
