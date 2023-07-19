package clustersetup

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argoAppSet "github.com/argoproj/applicationset/pkg/utils"
	"github.com/kubernetes-client/go-base/config/api"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	ocm "github.com/stolostron/cluster-templates-operator/ocm"
	utils "github.com/stolostron/cluster-templates-operator/utils"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterConfig struct {
	BearerToken     string          `json:"bearerToken"`
	TLSClientConfig TLSClientConfig `json:"tlsClientConfig"`
}

type TLSClientConfig struct {
	CAData string `json:"caData"`
}

type LoginError struct {
	Msg string
}

func (l *LoginError) Error() string {
	return l.Msg
}

func AddClusterToArgo(
	ctx context.Context,
	k8sClient client.Client,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
	getNewClusterClient func(configBytes []byte) (client.Client, error),
	argoCDNamespace string,
	withManagedCluster bool,
	loginAttemptTimeout time.Duration,
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

	if clusterTemplateInstance.Status.FirstLoginAttempt == nil {
		time := metav1.Now()
		clusterTemplateInstance.Status.FirstLoginAttempt = &time
	}
	newClusterClient, err := getNewClusterClient(kubeconfigSecret.Data["kubeconfig"])

	if err != nil {
		if time.Now().Before(clusterTemplateInstance.Status.FirstLoginAttempt.Add(loginAttemptTimeout)) {
			loginError := &LoginError{Msg: err.Error()}
			return loginError
		}
		return err
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-manager",
			Namespace: "kube-system",
		},
	}

	if err = utils.EnsureResourceExists(ctx, newClusterClient, sa, false); err != nil {
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

	if err = utils.EnsureResourceExists(ctx, newClusterClient, clusterRole, false); err != nil {
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

	if err = utils.EnsureResourceExists(ctx, newClusterClient, clusterRoleBinding, false); err != nil {
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

	if err = utils.EnsureResourceExists(ctx, newClusterClient, tokenSecret, true); err != nil {
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

	var appName string
	if clusterTemplateInstance.Spec.KubeconfigSecretRef != nil {
		appName = *clusterTemplateInstance.Spec.KubeconfigSecretRef
	} else {
		app, err := clusterTemplateInstance.GetDay1Application(ctx, k8sClient, argoCDNamespace)
		if err != nil {
			return err
		}
		appName = app.Name
	}

	clusterName := clusterTemplateInstance.Namespace + "/" + clusterTemplateInstance.Name
	if withManagedCluster {
		// ArgoCD cluster has to match MC name
		mc, err := ocm.GetManagedCluster(ctx, k8sClient, clusterTemplateInstance)
		if err != nil {
			return err
		}
		clusterName = mc.Name
	}

	clusterSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: argoCDNamespace,
			Labels: map[string]string{
				argoAppSet.ArgoCDSecretTypeLabel: argoAppSet.ArgoCDSecretTypeCluster,
				v1alpha1.CTINameLabel:            clusterTemplateInstance.Name,
				v1alpha1.CTINamespaceLabel:       clusterTemplateInstance.Namespace,
			},
		},
		Data: map[string][]byte{
			"name":   []byte(clusterName),
			"server": []byte(kubeconfig.Clusters[0].Cluster.Server),
			"config": jsonConfig,
		},
		Type: corev1.SecretTypeOpaque,
	}

	return utils.EnsureResourceExists(ctx, k8sClient, clusterSecret, false)
}

func GetClientForCluster(configBytes []byte) (client.Client, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(configBytes)

	if err != nil {
		return nil, err
	}

	return client.New(restConfig, client.Options{})
}
