package testutils

import (
	"encoding/json"
	"io/ioutil"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ctiName = "mycluster"
	ctiNs   = "default"
	ctName  = "mytemplate"
)

func GetCTI() *v1alpha1.ClusterTemplateInstance {
	cti := &v1alpha1.ClusterTemplateInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.APIVersion,
			Kind:       "ClusterTemplateInstance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctiName,
			Namespace: ctiNs,
			Finalizers: []string{
				v1alpha1.CTIFinalizer,
			},
		},
		Spec: v1alpha1.ClusterTemplateInstanceSpec{
			ClusterTemplateRef: ctName,
		},
	}
	return cti
}

func GetCT(withSetup bool) *v1alpha1.ClusterTemplate {
	ct := &v1alpha1.ClusterTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.Identifier(),
			Kind:       "ClusterTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ctName,
		},
		Spec: v1alpha1.ClusterTemplateSpec{
			ClusterDefinition: argo.ApplicationSpec{
				Source: argo.ApplicationSource{
					RepoURL:        "http://foo.com",
					TargetRevision: "0.1.0",
					Chart:          "hypershift-template",
				},
				Destination: argo.ApplicationDestination{
					Server:    "https://kubernetes.default.svc",
					Namespace: "default",
				},
				Project: "",
				SyncPolicy: &argo.SyncPolicy{
					Automated: &argo.SyncPolicyAutomated{},
				},
			},
		},
	}
	if withSetup {
		ct.Spec.ClusterSetup = []v1alpha1.ClusterSetup{
			{
				Name: "day2",
				Spec: argo.ApplicationSpec{
					Source: argo.ApplicationSource{
						RepoURL:        "http://foo.com",
						TargetRevision: "0.1.0",
						Chart:          "day2-template",
					},
					Destination: argo.ApplicationDestination{
						Server:    "${new_cluster}",
						Namespace: "default",
					},
					Project: "",
					SyncPolicy: &argo.SyncPolicy{
						Automated: &argo.SyncPolicyAutomated{},
					},
				},
			},
		}
	}
	return ct
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

func SetHostedClusterReady(
	hostedCluster hypershiftv1alpha1.HostedCluster,
	kubeconfigName string,
	kubeadminName string,
) hypershiftv1alpha1.HostedCluster {
	status := hypershiftv1alpha1.HostedClusterStatus{}

	status.Conditions = []metav1.Condition{
		{
			Type:               string(hypershiftv1alpha1.HostedClusterAvailable),
			Status:             metav1.ConditionTrue,
			Reason:             "Foo",
			LastTransitionTime: metav1.Now(),
		},
	}
	status.KubeConfig = &corev1.LocalObjectReference{
		Name: kubeconfigName,
	}
	status.KubeadminPassword = &corev1.LocalObjectReference{
		Name: kubeadminName,
	}
	hostedCluster.Status = status
	hostedCluster.Labels = map[string]string{
		"foo": "bar",
	}
	return hostedCluster
}
