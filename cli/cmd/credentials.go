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

var ocLogin = "false"

type CredentialsOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
	Namespace string
}

func NewCredentialsOptions(namespace string, streams genericclioptions.IOStreams) *CredentialsOptions {
	return &CredentialsOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
		Namespace:   namespace,
	}
}

func NewCmdCredentials(k8sClient client.Client, namespace string, streams genericclioptions.IOStreams) *cobra.Command {
	o := NewCredentialsOptions(namespace, streams)
	cmd := &cobra.Command{
		Use:          "credentials [cluster-name]",
		Short:        "Get cluster credentials",
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

func (cd *CredentialsOptions) run(k8sClient client.Client, args []string) error {
	cti := &v1alpha1.ClusterTemplateInstance{}
	if err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: args[0], Namespace: cd.Namespace}, cti); err != nil {
		return err
	}

	secret := &corev1.Secret{}
	if err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: cti.Status.AdminPassword.Name, Namespace: "devuserns"}, secret); err != nil {
		return err
	}

	username := string(secret.Data["username"])
	password := string(secret.Data["password"])

	result := fmt.Sprintf("Username: %s\nPassword: %s\nAPI URL: %s\n",
		username,
		password,
		cti.Status.APIserverURL,
	)

	if ocLogin == "true" {
		result = result + fmt.Sprintf("Login cmd: oc login %s -u %s -p %s\n",
			cti.Status.APIserverURL,
			username,
			password,
		)
	}

	_, err := fmt.Fprint(cd.Out, result)
	if err != nil {
		return err
	}
	return nil
}
