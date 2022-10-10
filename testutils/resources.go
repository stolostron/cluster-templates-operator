package testutils

import (
	"encoding/json"
	"io/ioutil"

	openshiftAPI "github.com/openshift/api/helm/v1beta1"
	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
)

const (
	ctiName = "mycluster"
	ctiNs   = "default"
	ctName  = "mytemplate"

	helmRepoName = "myrepo"

	pipelineName = "mypipeline"
	pipelineNs   = "pipelines"
)

func GetCTI() (*v1alpha1.ClusterTemplateInstance, types.NamespacedName) {
	cti := &v1alpha1.ClusterTemplateInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.APIVersion,
			Kind:       "ClusterTemplateInstance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctiName,
			Namespace: ctiNs,
		},
		Spec: v1alpha1.ClusterTemplateInstanceSpec{
			ClusterTemplateRef: ctName,
		},
	}
	return cti, types.NamespacedName{Name: cti.Name, Namespace: cti.Namespace}
}

func GetCT(withProps bool, withSetup bool) (*v1alpha1.ClusterTemplate, error) {
	ct := &v1alpha1.ClusterTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       "ClusterTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ctName,
		},
		Spec: v1alpha1.ClusterTemplateSpec{
			HelmChartRef: &v1alpha1.HelmChartRef{
				Repository: helmRepoName,
				Name:       "hypershift-template",
				Version:    "0.1.0",
			},
		},
	}
	if withProps {
		props, err := GetHypershiftTemplateProps()
		if err != nil {
			return nil, err
		}
		ct.Spec.Properties = props
	}
	if withSetup {
		ct.Spec.ClusterSetup = &v1alpha1.ClusterSetup{
			Pipeline: v1alpha1.ResourceRef{
				Name:      pipelineName,
				Namespace: pipelineNs,
			},
		}
	}
	return ct, nil
}

func GetHelmRepo(serverURL string) *openshiftAPI.HelmChartRepository {
	helmRepoCR := &openshiftAPI.HelmChartRepository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: openshiftAPI.GroupVersion.Identifier(),
			Kind:       "HelmChartRepository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: helmRepoName,
		},
		Spec: openshiftAPI.HelmChartRepositorySpec{
			ConnectionConfig: openshiftAPI.ConnectionConfig{
				URL: serverURL,
			},
		},
	}
	return helmRepoCR
}

func GetKubeconfigSecret() (*corev1.Secret, error) {
	kubeconfigFile, err := ioutil.ReadFile("../testutils/kubeconfig_mock.yaml")
	if err != nil {
		return nil, err
	}
	kubeConfigSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hypershift-kube-config",
			Namespace: ctiNs,
		},
		Data: map[string][]byte{
			"kubeconfig": kubeconfigFile,
		},
	}
	return kubeConfigSecret, nil
}

func GetKubeadminSecret() (*corev1.Secret, error) {
	kubeAdminBytes, err := json.Marshal("foo")
	if err != nil {
		return nil, err
	}
	kubeAdminSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hypershift-kube-admin-pass",
			Namespace: ctiNs,
		},
		Data: map[string][]byte{
			"password": kubeAdminBytes,
		},
	}
	return kubeAdminSecret, nil
}

func GetHypershiftTemplateProps() ([]v1alpha1.Property, error) {
	psVal, err := json.Marshal("eyJmb28iOiAibXlwdWxsc2VjcmV0In0=")
	if err != nil {
		return []v1alpha1.Property{}, err
	}
	sshVal, err := json.Marshal("bXlzc2hrZXk=")
	if err != nil {
		return []v1alpha1.Property{}, err
	}
	return []v1alpha1.Property{
		{
			Name:         "pullSecret",
			Description:  "pullSecret",
			Type:         v1alpha1.PropertyTypeString,
			DefaultValue: psVal,
		},
		{
			Name:         "sshPublicKey",
			Description:  "sshPublicKey",
			Type:         v1alpha1.PropertyTypeString,
			DefaultValue: sshVal,
		},
	}, nil
}

func SetHostedClusterReady(hostedCluster *hypershiftv1alpha1.HostedCluster, kubeconfigName string, kubeadminName string) {
	hostedCluster.Status = hypershiftv1alpha1.HostedClusterStatus{
		Conditions: []metav1.Condition{
			{
				Type:               string(hypershiftv1alpha1.HostedClusterAvailable),
				Status:             metav1.ConditionTrue,
				Reason:             "Foo",
				LastTransitionTime: metav1.Now(),
			},
		},
		KubeConfig: &corev1.LocalObjectReference{
			Name: kubeconfigName,
		},
		KubeadminPassword: &corev1.LocalObjectReference{
			Name: kubeadminName,
		},
	}
}

func GetClusterTask() *tekton.ClusterTask {
	clusterTask := &tekton.ClusterTask{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "ClusterTask",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-task",
		},
		Spec: tekton.TaskSpec{
			Steps: []tekton.Step{
				{
					Image: "registry.redhat.io/ubi7/ubi-minimal",
					Command: []string{
						"/bin/bash",
						"'-c'",
						"echo",
						"foo",
					},
				},
			},
		},
	}
	return clusterTask
}

func GetPipeline() (*tekton.Pipeline, *corev1.Namespace) {
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pipelineNs,
		},
	}
	pipeline := &tekton.Pipeline{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Pipeline",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineName,
			Namespace: pipelineNs,
		},
		Spec: tekton.PipelineSpec{
			Tasks: []tekton.PipelineTask{
				{
					Name: "footask",
					TaskRef: &tekton.TaskRef{
						Kind: tekton.ClusterTaskKind,
						Name: "cluster-task",
					},
				},
			},
			Workspaces: []tekton.PipelineWorkspaceDeclaration{
				{
					Name: "kubeconfigSecret",
				},
				{
					Name: "kubeadminPassSecret",
				},
			},
		},
	}
	return pipeline, ns
}

func SetPipelineRunSucceeded(pipelineRun *tekton.PipelineRun) {
	pipelineRun.Status.Conditions = []apis.Condition{
		{
			Type:   "Succeeded",
			Status: corev1.ConditionTrue,
			Reason: "Foo",
			LastTransitionTime: apis.VolatileTime{
				Inner: metav1.Now(),
			},
		},
	}
}
