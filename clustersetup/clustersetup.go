package clustersetup

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argoAppSet "github.com/argoproj/applicationset/pkg/utils"
	"github.com/kubernetes-client/go-base/config/api"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterConfig struct {
	BearerToken     string          `json:"bearerToken"`
	TLSClientConfig TLSClientConfig `json:"tlsClientConfig"`
}

type TLSClientConfig struct {
	CAData string `json:"caData"`
}

func AddClusterToArgo(
	ctx context.Context,
	k8sClient client.Client,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	getNewClusterClient func(configBytes []byte) (client.Client, error),
	argoCDNamespace string,
) error {
	kubeconfigSecret := corev1.Secret{}

	if err := k8sClient.Get(
		ctx,
		client.ObjectKey{
			Name:      clusterTemplateInstance.GetKubeconfigRef(),
			Namespace: clusterTemplateInstance.Namespace,
		},
		&kubeconfigSecret,
	); err != nil {
		return err
	}

	newClusterClient, err := getNewClusterClient(kubeconfigSecret.Data["kubeconfig"])

	if err != nil {
		return err
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-manager",
			Namespace: "kube-system",
		},
	}

	if err = ensureResourceExists(ctx, newClusterClient, sa, false); err != nil {
		return err
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: sa.Name + "-role",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"*"},
				APIGroups: []string{"*"},
				Resources: []string{"*"},
			},
			{
				NonResourceURLs: []string{"*"},
				Verbs:           []string{"*"},
			},
		},
	}

	if err = ensureResourceExists(ctx, newClusterClient, clusterRole, false); err != nil {
		return err
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: sa.Name + "-role-binding",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		},
	}

	if err = ensureResourceExists(ctx, newClusterClient, clusterRoleBinding, false); err != nil {
		return err
	}

	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sa.Name + "-token",
			Namespace: sa.Namespace,
			Annotations: map[string]string{
				corev1.ServiceAccountNameKey: sa.Name,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}

	if err = ensureResourceExists(ctx, newClusterClient, tokenSecret, true); err != nil {
		return err
	}

	if len(tokenSecret.Data["token"]) == 0 {
		return fmt.Errorf("token not found")
	}
	if len(tokenSecret.Data["ca.crt"]) == 0 {
		return fmt.Errorf("ca.crt not found")
	}

	kubeconfig := api.Config{}
	if err := yaml.Unmarshal(kubeconfigSecret.Data["kubeconfig"], &kubeconfig); err != nil {
		return err
	}

	config := ClusterConfig{
		BearerToken: string(tokenSecret.Data["token"]),
		TLSClientConfig: TLSClientConfig{
			CAData: base64.URLEncoding.EncodeToString(tokenSecret.Data["ca.crt"]),
		},
	}

	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return err
	}

	app, err := clusterTemplateInstance.GetDay1Application(ctx, k8sClient, argoCDNamespace)
	if err != nil {
		return err
	}

	clusterSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels: map[string]string{
				argoAppSet.ArgoCDSecretTypeLabel: argoAppSet.ArgoCDSecretTypeCluster,
				v1alpha1.CTINameLabel:            clusterTemplateInstance.Name,
				v1alpha1.CTINamespaceLabel:       clusterTemplateInstance.Namespace,
			},
		},
		StringData: map[string]string{
			"name":   clusterTemplateInstance.Namespace + "/" + clusterTemplateInstance.Name,
			"server": kubeconfig.Clusters[0].Cluster.Server,
			"config": string(jsonConfig),
		},
		Type: corev1.SecretTypeOpaque,
	}

	return ensureResourceExists(ctx, k8sClient, clusterSecret, false)
}

func ensureResourceExists(
	ctx context.Context,
	newClusterClient client.Client,
	obj client.Object,
	loadBack bool,
) error {
	if err := newClusterClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if apierrors.IsNotFound(err) {
			if err = newClusterClient.Create(ctx, obj); err != nil {
				return err
			}
			if loadBack {
				if err = newClusterClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}
	return nil
}

func GetClientForCluster(configBytes []byte) (client.Client, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(configBytes)

	if err != nil {
		return nil, err
	}

	return client.New(restConfig, client.Options{})
}
