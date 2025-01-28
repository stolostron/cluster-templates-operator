package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	argoOperator "github.com/argoproj-labs/argocd-operator/api/v1alpha1"
	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	mce "github.com/stolostron/backplane-operator/api/v1"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	addonapi "open-cluster-management.io/api/addon/v1alpha1"
	ocm "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme = runtime.NewScheme()
)

func NewCmdRoot(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "cluster",
		Short:        "Cluster-aaS kubectl plugin",
		SilenceUsage: true,
	}

	k8sClient, ns := CreateK8sClient()
	cmd.AddCommand(NewCmdCredentials(k8sClient, ns, streams))
	cmd.AddCommand(NewCmdKubeconfig(k8sClient, ns, streams))
	cmd.AddCommand(NewCmdTemplates(k8sClient, ns, streams))
	cmd.AddCommand(NewCmdTemplateDescribe(k8sClient, streams))
	cmd.AddCommand(NewCmdListInstances(k8sClient, ns, streams))
	cmd.AddCommand(NewCmdInstallOperator(k8sClient, streams))
	cmd.AddCommand(NewCmdUninstallOperator(k8sClient, streams))
	return cmd
}

func CreateK8sClient() (client.Client, string) {
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(olm.AddToScheme(scheme))
	utilruntime.Must(olmv1.AddToScheme(scheme))
	utilruntime.Must(argoOperator.AddToScheme(scheme))
	utilruntime.Must(mce.AddToScheme(scheme))
	utilruntime.Must(ocm.AddToScheme(scheme))
	utilruntime.Must(addonapi.AddToScheme(scheme))
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		os.Exit(1)
	}

	k8sClient, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		os.Exit(1)
	}
	ns, _, err := kubeConfig.Namespace()
	if err != nil {
		os.Exit(1)
	}
	return k8sClient, ns
}
