package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TemplatesOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
	Namespace string
}

// NewNamespaceOptions provides an instance of NamespaceOptions with default values
func NewTemplatesOptions(namespace string, streams genericclioptions.IOStreams) *TemplatesOptions {
	return &TemplatesOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
		Namespace:   namespace,
	}
}

func NewCmdTemplates(
	k8sClient client.Client,
	namespace string,
	streams genericclioptions.IOStreams,
) *cobra.Command {
	o := NewTemplatesOptions(namespace, streams)
	cmd := &cobra.Command{
		Use:          "templates",
		Short:        "View cluster templates and quotas",
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

func (sv *TemplatesOptions) run(k8sClient client.Client, args []string) error {
	cts := &v1alpha1.ClusterTemplateList{}
	if err := k8sClient.List(context.TODO(), cts); err != nil {
		return err
	}

	ctqs := &v1alpha1.ClusterTemplateQuotaList{}
	if err := k8sClient.List(context.TODO(), ctqs, &client.ListOptions{Namespace: sv.Namespace}); err != nil {
		return err
	}

	w := tabwriter.NewWriter(sv.Out, 10, 1, 5, ' ', 0)
	fsHeader := "%s\t%s\t%s\t%s\n"
	fs := "%s\t%t\t%s\t%s\n"
	fmt.Fprintf(w, fsHeader, "NAME", "ALLOWED", "USED/MAX", "COST/BUDGET")
	for _, ct := range cts.Items {
		ctq := ctqs.Items[0]
		allowed := false
		max := "-"
		used := 0
		for _, allowedTemplate := range ctq.Spec.AllowedTemplates {
			if allowedTemplate.Name == ct.Name {
				allowed = true
				if allowedTemplate.Count > 0 {
					max = fmt.Sprint(allowedTemplate.Count)
				}
			}
			for _, templateStatus := range ctq.Status.TemplateInstances {
				if templateStatus.Name == ct.Name {
					used = templateStatus.Count
				}
			}
		}

		if !allowed {
			fmt.Fprintf(w, fs, ct.Name, allowed, "-", "-")
		} else {
			budget := "-"
			if ctq.Spec.Budget != 0 {
				budget = fmt.Sprint(ctq.Spec.Budget)
			}

			fmt.Fprintf(w, fs, ct.Name, allowed, fmt.Sprint(used)+"/"+fmt.Sprint(max), fmt.Sprint(ct.Spec.Cost)+"/"+budget)
		}

	}

	return w.Flush()
}
