package clusterprovider

import (
	"context"
	"encoding/base64"
	"errors"

	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	v1alpha1 "github.com/rawagner/cluster-templates-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HostedClusterProvider struct {
	HostedCluster string
}

func (hc HostedClusterProvider) GetClusterStatus(
	ctx context.Context,
	k8sClient client.Client,
	templateInstance v1alpha1.ClusterTemplateInstance,
) (bool, string, error) {
	var hostedC hypershiftv1alpha1.HostedCluster
	err := k8sYaml.Unmarshal([]byte(hc.HostedCluster), &hostedC)
	if err != nil {
		return false, "", err
	}

	hostedCluster := &hypershiftv1alpha1.HostedCluster{}
	err = k8sClient.Get(
		ctx,
		client.ObjectKey{Name: hostedC.Name, Namespace: hostedC.Namespace},
		hostedCluster,
	)

	if err != nil {
		return false, "", err
	}

	for _, condition := range hostedCluster.Status.Conditions {
		if condition.Type == string(hypershiftv1alpha1.HostedClusterAvailable) {
			if condition.Status == metav1.ConditionTrue {
				hypershiftPass := getKubeAdminRef(*hostedCluster)
				hypershiftKubeConfig := getKubeConfigRef(*hostedCluster)

				if hypershiftPass == "" || hypershiftKubeConfig == "" {
					return false, "Waiting for pass/kubeconfig secrets", nil
				}

				hypershiftKubeconfigSecret := corev1.Secret{}
				err := k8sClient.Get(
					ctx,
					client.ObjectKey{
						Name:      hypershiftKubeConfig,
						Namespace: hostedCluster.Namespace,
					},
					&hypershiftKubeconfigSecret,
				)

				if err != nil {
					return false, "", err
				}

				kubeconfigBytes, ok := hypershiftKubeconfigSecret.Data["kubeconfig"]

				if !ok {
					return false, "", errors.New("unexpected kubeconfig format")
				}

				hypershiftKubeadminSecret := corev1.Secret{}
				err = k8sClient.Get(
					ctx,
					client.ObjectKey{Name: hypershiftPass, Namespace: hostedCluster.Namespace},
					&hypershiftKubeadminSecret,
				)

				if err != nil {
					return false, "", err
				}

				kubeadminPass, ok := hypershiftKubeadminSecret.Data["password"]

				if !ok {
					return false, "", errors.New("unexpected kubeadmin password format")
				}

				username := base64.URLEncoding.EncodeToString([]byte("kubeadmin"))

				err = CreateClusterSecrets(
					ctx,
					k8sClient,
					kubeconfigBytes,
					[]byte(username),
					kubeadminPass,
					templateInstance,
				)
				if err != nil {
					return false, "", err
				}
				return true, "Available", nil
			} else {
				return false, "Not available - " + condition.Reason, nil
			}
		}
	}

	return false, "Not available", nil
}

func getKubeAdminRef(hostedCluster hypershiftv1alpha1.HostedCluster) string {
	if hostedCluster.Status.KubeadminPassword != nil {
		return hostedCluster.Status.KubeadminPassword.Name
	}
	return ""
}

func getKubeConfigRef(hostedCluster hypershiftv1alpha1.HostedCluster) string {
	if hostedCluster.Status.KubeConfig != nil {
		return hostedCluster.Status.KubeConfig.Name
	}
	return ""
}
