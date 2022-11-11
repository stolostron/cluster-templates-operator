package clusterprovider

import (
	"context"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"
	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterProviderExperimentalAnnotation = "clustertemplate.openshift.io/experimental-provider"
)

type ClusterProvider interface {
	GetClusterStatus(
		ctx context.Context,
		k8sClient client.Client,
		templateInstance v1alpha1.ClusterTemplateInstance,
	) (bool, string, error)
}

func GetClusterProvider(application argo.Application, log logr.Logger) ClusterProvider {
	for _, obj := range application.Status.Resources {
		switch obj.Kind {
		case "HostedCluster":
			log.Info("Cluster provider: HostedCluster")
			if obj.Version != "v1alpha1" {
				log.Info("Unknown version: ", obj.Version)
				return nil
			}
			nodePools := []string{}
			for _, obj := range application.Status.Resources {
				if obj.Kind == "NodePool" {
					nodePools = append(nodePools, obj.Name)
				}
			}
			return HostedClusterProvider{
				HostedClusterName:      obj.Name,
				HostedClusterNamespace: obj.Namespace,
				NodePoolNames:          nodePools,
			}
		case "ClusterDeployment":
			log.Info("Cluster provider: ClusterDeployment")
			if obj.Version != "v1" {
				log.Info("Unknown version: ", obj.Version)
				return nil
			}
			return ClusterDeploymentProvider{
				ClusterDeploymentName:      obj.Name,
				ClusterDeploymentNamespace: obj.Namespace,
			}
		case "ClusterClaim":
			log.Info("Cluster provider: ClusterClaim")
			if obj.Version != "v1" {
				log.Info("Unknown version: ", obj.Version)
				return nil
			}
			return ClusterClaimProvider{
				ClusterClaimName:      obj.Name,
				ClusterClaimNamespace: obj.Namespace,
			}
		}
	}
	log.Info("Cluster provider: Unknown")
	return nil
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

	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&kubeconfigSecret), &kubeconfigSecret); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		kubeconfigSecret.Data = map[string][]byte{
			"kubeconfig": kubeconfig,
		}
		kubeconfigSecret.OwnerReferences = []metav1.OwnerReference{
			templateInstance.GetOwnerReference(),
		}

		if err := k8sClient.Create(ctx, &kubeconfigSecret); err != nil {
			return err
		}
	}

	kubeadminSecret := corev1.Secret{}
	kubeadminSecret.Name = templateInstance.GetKubeadminPassRef()
	kubeadminSecret.Namespace = templateInstance.Namespace

	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&kubeadminSecret), &kubeadminSecret); err != nil {
		if apierrors.IsNotFound(err) {
			kubeadminSecret.Data = map[string][]byte{
				"username": kubeadmin,
				"password": kubeadminpass,
			}
			kubeadminSecret.OwnerReferences = []metav1.OwnerReference{
				templateInstance.GetOwnerReference(),
			}

			if err = k8sClient.Create(ctx, &kubeadminSecret); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
