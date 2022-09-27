package clusterprovider

import (
	"context"
	"strings"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/release"

	"github.com/go-logr/logr"
	v1alpha1 "github.com/rawagner/cluster-templates-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterProvider interface {
	GetClusterStatus(
		ctx context.Context,
		k8sClient client.Client,
		templateInstance v1alpha1.ClusterTemplateInstance,
	) (bool, string, error)
}

func GetClusterProvider(helmRelease *release.Release, log logr.Logger) (ClusterProvider, error) {
	stringObjects := strings.Split(helmRelease.Manifest, "---\n")

	for _, obj := range stringObjects {
		var yamlObj map[string]interface{}
		err := yaml.Unmarshal([]byte(obj), &yamlObj)
		if err != nil {
			return nil, err
		}
		switch yamlObj["kind"] {
		case "HostedCluster":
			log.Info("Cluster provider: HostedCluster")
			return HostedClusterProvider{HostedCluster: obj}, nil
		case "ClusterDeployment":
			log.Info("Cluster provider: ClusterDeployment")
			return ClusterDeploymentProvider{ClusterDeployment: obj}, nil
		case "ClusterClaim":
			log.Info("Cluster provider: ClusterClaim")
			return ClusterClaimProvider{ClusterClaim: obj}, nil
		}
	}
	log.Info("Cluster provider: Unknown")
	return nil, nil
}

func CreateClusterSecrets(
	ctx context.Context,
	k8sClient client.Client,
	kubeconfig []byte,
	kubeadmin []byte,
	kubeadminpass []byte,
	templateInstance v1alpha1.ClusterTemplateInstance,
) error {
	kubeconfigSecret := corev1.Secret{}
	kubeconfigSecret.Name = templateInstance.GetKubeconfigRef()
	kubeconfigSecret.Namespace = templateInstance.Namespace

	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&kubeconfigSecret), &kubeconfigSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			kubeconfigSecret.Data = map[string][]byte{
				"kubeconfig": kubeconfig,
			}
			kubeconfigSecret.OwnerReferences = []metav1.OwnerReference{
				{
					Kind:       "ClusterTemplateInstance",
					APIVersion: v1alpha1.APIVersion,
					Name:       templateInstance.Name,
					UID:        templateInstance.UID,
				},
			}

			err := k8sClient.Create(ctx, &kubeconfigSecret)

			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	kubeadminSecret := corev1.Secret{}
	kubeadminSecret.Name = templateInstance.GetKubeadminPassRef()
	kubeadminSecret.Namespace = templateInstance.Namespace

	err = k8sClient.Get(ctx, client.ObjectKeyFromObject(&kubeadminSecret), &kubeadminSecret)

	if err != nil {
		if apierrors.IsNotFound(err) {
			kubeadminSecret.Data = map[string][]byte{
				"username": kubeadmin,
				"password": kubeadminpass,
			}
			kubeadminSecret.OwnerReferences = []metav1.OwnerReference{
				{
					Kind:       "ClusterTemplateInstance",
					APIVersion: v1alpha1.APIVersion,
					Name:       templateInstance.Name,
					UID:        templateInstance.UID,
				},
			}

			err = k8sClient.Create(ctx, &kubeadminSecret)

			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
