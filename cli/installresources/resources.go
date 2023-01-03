package installresources

import (
	"os"

	argoOperator "github.com/argoproj-labs/argocd-operator/api/v1alpha1"
	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	mce "github.com/stolostron/backplane-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	addonapi "open-cluster-management.io/api/addon/v1alpha1"
	ocm "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorNamespace = "cluster-aas-operator"
	argocdNamespace   = "argocd"
	templateNamespace = "clusters"
	mceNamespace      = "multicluster-engine"
)

var (
	OperatorNs = &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: operatorNamespace,
		},
	}
	OperatorGroup = &olmv1.OperatorGroup{
		ObjectMeta: v1.ObjectMeta{
			Name:      "cluster-aas-operator",
			Namespace: operatorNamespace,
		},
	}
	OperatorSubscription = &olm.Subscription{
		ObjectMeta: v1.ObjectMeta{
			Name:      "cluster-aas-operator",
			Namespace: operatorNamespace,
		},
		Spec: &olm.SubscriptionSpec{
			Package:                "cluster-aas-operator",
			Channel:                "alpha",
			InstallPlanApproval:    "Automatic",
			CatalogSource:          "community-operators",
			CatalogSourceNamespace: "openshift-marketplace",
		},
	}
	ArgoNs = &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: argocdNamespace,
		},
	}
	ArgoInstance = &argoOperator.ArgoCD{
		ObjectMeta: v1.ObjectMeta{
			Name:      "argocd-sample",
			Namespace: argocdNamespace,
		},
		Spec: argoOperator.ArgoCDSpec{
			Controller: argoOperator.ArgoCDApplicationControllerSpec{
				Resources: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
						corev1.ResourceMemory: resource.MustParse("2048Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(250, resource.DecimalSI),
						corev1.ResourceMemory: resource.MustParse("1024Mi"),
					},
				},
			},
			HA: argoOperator.ArgoCDHASpec{
				Enabled: true,
				Resources: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(500, resource.DecimalSI),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(250, resource.DecimalSI),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
				},
			},
			Redis: argoOperator.ArgoCDRedisSpec{
				Resources: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(500, resource.DecimalSI),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(250, resource.DecimalSI),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
				},
			},
			Repo: argoOperator.ArgoCDRepoSpec{
				Resources: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(250, resource.DecimalSI),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			},
			Server: argoOperator.ArgoCDServerSpec{
				Resources: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(500, resource.DecimalSI),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(125, resource.DecimalSI),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
				},
				Route: argoOperator.ArgoCDRouteSpec{
					Enabled: true,
				},
			},
		},
	}
	ClusterNs = &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: templateNamespace,
			Labels: map[string]string{
				"argocd.argoproj.io/managed-by": "argocd",
			},
		},
	}
	PullSecret = &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "pullsecret-cluster",
			Namespace: templateNamespace,
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte(os.Getenv("PULL_SECRET")),
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}
	SshKeySecret = &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "sshkey-cluster",
			Namespace: templateNamespace,
		},
		Data: map[string][]byte{
			"id_rsa.pub": []byte(os.Getenv("SSH_KEY")),
		},
	}
	MceNs = &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: mceNamespace,
		},
	}
	MceOperatorGroup = &olmv1.OperatorGroup{
		ObjectMeta: v1.ObjectMeta{
			Name:      "multicluster-engine",
			Namespace: mceNamespace,
		},
		Spec: olmv1.OperatorGroupSpec{
			TargetNamespaces: []string{mceNamespace},
		},
	}
	MceSub = &olm.Subscription{
		ObjectMeta: v1.ObjectMeta{
			Name:      "multicluster-engine",
			Namespace: mceNamespace,
		},
		Spec: &olm.SubscriptionSpec{
			Channel:                "stable-2.1",
			InstallPlanApproval:    "Automatic",
			Package:                "multicluster-engine",
			CatalogSource:          "redhat-operators",
			CatalogSourceNamespace: "openshift-marketplace",
		},
	}
	Mce = &mce.MultiClusterEngine{
		ObjectMeta: v1.ObjectMeta{
			Name: "multiclusterengine",
		},
		Spec: mce.MultiClusterEngineSpec{
			AvailabilityConfig: mce.HAHigh,
			TargetNamespace:    mceNamespace,
			Overrides: &mce.Overrides{
				Components: []mce.ComponentConfig{
					{
						Name:    "hypershift-preview",
						Enabled: true,
					},
				},
			},
		},
	}
	MceManagedCluster = &ocm.ManagedCluster{
		ObjectMeta: v1.ObjectMeta{
			Name: "local-cluster",
			Labels: map[string]string{
				"local-cluster": "true",
				"cloud":         "auto-detect",
				"vendor":        "auto-detect",
			},
		},
		Spec: ocm.ManagedClusterSpec{
			HubAcceptsClient: true,
		},
	}
	MceHypershiftAddon = &addonapi.ManagedClusterAddOn{
		ObjectMeta: v1.ObjectMeta{
			Name:      "hypershift-addon",
			Namespace: "local-cluster",
		},
		Spec: addonapi.ManagedClusterAddOnSpec{
			InstallNamespace: "open-cluster-management-agent-addon",
		},
	}
)

type InstallRes struct {
	Key    string
	Object client.Object
}

var InstallResources = []client.Object{
	OperatorNs,
	OperatorGroup,
	OperatorSubscription,
	ArgoNs,
	ArgoInstance,
	ClusterNs,
	PullSecret,
	SshKeySecret,
	MceNs,
	MceOperatorGroup,
	MceSub,
	Mce,
	MceManagedCluster,
	MceHypershiftAddon,
}
