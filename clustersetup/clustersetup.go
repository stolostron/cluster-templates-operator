package clustersetup

import (
	"context"
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClusterSetupInstanceLabel = "clustertemplate.openshift.io/cluster-instance"
)

func CreateSetupPipeline(
	ctx context.Context,
	log logr.Logger,
	k8sClient client.Client,
	clusterTemplate v1alpha1.ClusterTemplate,
	clusterTemplateInstance *v1alpha1.ClusterTemplateInstance,
) error {
	pipelines := pipelinev1beta1.PipelineList{}

	clusterSetup := clusterTemplate.Spec.ClusterSetup

	if clusterSetup.Pipeline.Name != "" {
		if err := k8sClient.List(ctx, &pipelines, &client.ListOptions{}); err != nil {
			return err
		}
	}

	log.Info("Create PipelineRun")
	pipelineRun := &pipelinev1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: clusterTemplateInstance.Name + "-",
			Namespace:    clusterTemplateInstance.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				clusterTemplateInstance.GetOwnerReference(),
			},
			Labels: map[string]string{
				ClusterSetupInstanceLabel: clusterTemplateInstance.Name,
			},
		},
		Spec: pipelinev1beta1.PipelineRunSpec{
			Workspaces: []pipelinev1beta1.WorkspaceBinding{
				{
					Name: "kubeconfigSecret",
					Secret: &corev1.SecretVolumeSource{
						SecretName: clusterTemplateInstance.GetKubeconfigRef(),
					},
				},
				{
					Name: "kubeadminPassSecret",
					Secret: &corev1.SecretVolumeSource{
						SecretName: clusterTemplateInstance.GetKubeadminPassRef(),
					},
				},
			},
		},
	}

	values := make(map[string]interface{})
	if len(clusterTemplateInstance.Spec.Values) > 0 {
		if err := json.Unmarshal(clusterTemplateInstance.Spec.Values, &values); err != nil {
			return err
		}
	}

	clusterSetupParams := []pipelinev1beta1.Param{}
	for _, prop := range clusterTemplate.Spec.Properties {
		if prop.ClusterSetup {

			value := ""

			if len(prop.DefaultValue) > 0 {
				if err := json.Unmarshal(prop.DefaultValue, &value); err != nil {
					return err
				}
			}

			if prop.Overwritable {
				if val, ok := values[prop.Name]; ok {
					value = val.(string)
				}
			}

			clusterSetupParams = append(clusterSetupParams, pipelinev1beta1.Param{
				Name: prop.Name,
				Value: pipelinev1beta1.ArrayOrString{
					Type:      pipelinev1beta1.ParamTypeString,
					StringVal: value,
				},
			})
		}
	}

	pipelineRun.Spec.Params = clusterSetupParams

	if clusterSetup.Pipeline.Namespace != "" {
		var pipeline pipelinev1beta1.Pipeline
		for _, p := range pipelines.Items {
			if p.Name == clusterSetup.Pipeline.Name &&
				p.Namespace == clusterSetup.Pipeline.Namespace {
				pipeline = p
				break
			}
		}

		pipelineRun.Spec.PipelineSpec = &pipeline.Spec
	} else {
		pipelineRun.Spec.PipelineRef = &clusterSetup.PipelineRef
	}
	log.Info("Submit PipelineRun")
	return k8sClient.Create(ctx, pipelineRun)
}