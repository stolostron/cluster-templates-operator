package cmd

import (
	"context"
	"fmt"
	"strings"

	markdown "github.com/MichaelMure/go-term-markdown"
	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TemplateDescribeOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
}

func NewTemplateDescribeOptions(streams genericclioptions.IOStreams) *TemplateDescribeOptions {
	return &TemplateDescribeOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

func NewCmdTemplateDescribe(
	k8sClient client.Client,
	streams genericclioptions.IOStreams,
) *cobra.Command {
	o := NewTemplateDescribeOptions(streams)
	cmd := &cobra.Command{
		Use:          "template [template-name(s)]",
		Short:        "View cluster template(s) details",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := validate(args); err != nil {
				return err
			}
			if err := o.run(k8sClient, args); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func validate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("at least one template name is required")
	}

	return nil
}

func (td *TemplateDescribeOptions) run(k8sClient client.Client, args []string) error {
	cts := &v1alpha1.ClusterTemplateList{}
	if err := k8sClient.List(context.TODO(), cts); err != nil {
		return err
	}

	result := ""
	for index, templateName := range args {
		found := false
		for _, ct := range cts.Items {
			if ct.Name == templateName {
				found = true
				result = result + templateToDescription(ct)
			}
		}
		if !found {
			result = result + fmt.Sprintf("Cluster template '%s' not found\n", templateName)
		}
		if index != len(args)-1 {
			result = result + "\n\n"
		}
	}
	_, err := fmt.Fprintln(td.Out, result)
	return err
}

func templateToDescription(ct v1alpha1.ClusterTemplate) string {
	description := "No description provided"
	if ct.Annotations != nil && ct.Annotations[v1alpha1.CTDescriptionLabel] != "" {
		description = ct.Annotations[v1alpha1.CTDescriptionLabel]
	}
	descriptionResult := markdown.Render(description, 80, 6)

	result := fmt.Sprintf(
		"Name: %s\nDescription:\n\n%s\nCost: %d\n",
		ct.Name,
		string(descriptionResult),
		ct.Spec.Cost,
	)

	properties := "Properties:"
	cdValues := ct.Status.ClusterDefinition.Values
	cdSchema := ct.Status.ClusterDefinition.Schema
	if cdValues != "" || cdSchema != "" {
		properties = properties + "\n\tClusterDefinition:\n"
		if cdValues != "" {
			properties = properties + fmt.Sprintf(
				"\t\tValues:\n\t\t\t%s\n",
				strings.ReplaceAll(cdValues, "\n", "\n\t\t\t"),
			)
		}
		if cdSchema != "" {
			properties = properties + fmt.Sprintf(
				"\t\tSchema:\n\t\t\t%s\n",
				strings.ReplaceAll(cdSchema, "\n", "\n\t\t\t"),
			)
		}
	}
	if len(ct.Status.ClusterSetup) > 0 {
		properties = properties + "\tCluster Setup:\n"
		for _, clusterSetup := range ct.Status.ClusterSetup {
			values := clusterSetup.Values
			schema := clusterSetup.Values
			if values != "" {
				properties = properties + fmt.Sprintf(
					"\t\tValues:\n\t\t\t%s\n",
					strings.ReplaceAll(values, "\n", "\n\t\t\t"),
				)
			}
			if schema != "" {
				properties = properties + fmt.Sprintf(
					"\t\tSchema:\n\t\t\t%s\n",
					strings.ReplaceAll(schema, "\n", "\n\t\t\t"),
				)
			}
		}
	}
	if properties == "Properties:" {
		return ""
	}
	return result + properties
}
