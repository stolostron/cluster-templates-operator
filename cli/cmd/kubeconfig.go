package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeconfigOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
	Namespace string
}

func NewKubeconfigOptions(
	namespace string,
	streams genericclioptions.IOStreams,
) *KubeconfigOptions {
	return &KubeconfigOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
		Namespace:   namespace,
	}
}

func NewCmdKubeconfig(
	k8sClient client.Client,
	namespace string,
	streams genericclioptions.IOStreams,
) *cobra.Command {
	o := NewKubeconfigOptions(namespace, streams)
	cmd := &cobra.Command{
		Use:          "kubeconfig [cluster-name]",
		Short:        "Get cluster kubeconfig",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.run(k8sClient, args); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func (kc *KubeconfigOptions) run(k8sClient client.Client, args []string) error {
	cti := &v1alpha1.ClusterTemplateInstance{}
	if err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: args[0], Namespace: kc.Namespace}, cti); err != nil {
		return err
	}

	secret := &corev1.Secret{}
	if err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: cti.Status.Kubeconfig.Name, Namespace: kc.Namespace}, secret); err != nil {
		return err
	}

	_, err := fmt.Fprintln(kc.Out, string(secret.Data["kubeconfig"]))
	return err
}
