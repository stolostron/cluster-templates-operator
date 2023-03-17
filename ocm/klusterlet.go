package ocm

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	utils "github.com/stolostron/cluster-templates-operator/utils"
	agent "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateKlusterletAddonConfig(
	ctx context.Context,
	k8sClient client.Client,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	mc, err := GetManagedCluster(ctx, k8sClient, clusterTemplateInstance)
	if err != nil {
		return err
	}
	klusterlet := &agent.KlusterletAddonConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mc.Name,
			Namespace: mc.Name,
		},
		Spec: agent.KlusterletAddonConfigSpec{
			IAMPolicyControllerConfig: agent.KlusterletAddonAgentConfigSpec{
				Enabled: true,
			},
			SearchCollectorConfig: agent.KlusterletAddonAgentConfigSpec{
				Enabled: true,
			},
			PolicyController: agent.KlusterletAddonAgentConfigSpec{
				Enabled: true,
			},
			ApplicationManagerConfig: agent.KlusterletAddonAgentConfigSpec{
				Enabled: true,
			},
			CertPolicyControllerConfig: agent.KlusterletAddonAgentConfigSpec{
				Enabled: true,
			},
		},
	}
	return utils.EnsureResourceExists(ctx, k8sClient, klusterlet, false)
}
