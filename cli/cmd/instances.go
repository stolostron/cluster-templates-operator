package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstancesOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
	Namespace string
}

// NewInstancesOptions provides an instance of InstancesOptions with default values
func NewInstancesOptions(namespace string, streams genericclioptions.IOStreams) *InstancesOptions {
	return &InstancesOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
		Namespace:   namespace,
	}
}

func NewCmdListInstances(
	k8sClient client.Client,
	namespace string,
	streams genericclioptions.IOStreams,
) *cobra.Command {
	o := NewInstancesOptions(namespace, streams)
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "View cluster template instances",
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

func (sv *InstancesOptions) run(k8sClient client.Client, args []string) error {
	ctis := &v1alpha1.ClusterTemplateInstanceList{}
	if err := k8sClient.List(context.TODO(), ctis, &client.ListOptions{Namespace: sv.Namespace}); err != nil {
		return err
	}

	w := tabwriter.NewWriter(sv.Out, 10, 1, 5, ' ', 0)
	fs := "%s\t%s\t%s\t%s\t%s\n"
	fmt.Fprintf(w, fs, "NAME", "PHASE", "REQUESTER", "TEMPLATE", "AGE")
	for _, cti := range ctis.Items {
		requester := "-"
		if cti.Annotations != nil {
			req, ok := cti.Annotations[v1alpha1.CTIRequesterAnnotation]
			if ok {
				requester = req
			}
		}

		age := "<unknown>"
		timestamp := cti.CreationTimestamp
		if !timestamp.IsZero() {
			age = duration.HumanDuration(time.Since(timestamp.Time))
		}

		fmt.Fprintf(w, fs, cti.Name, cti.Status.Phase, requester, cti.Spec.ClusterTemplateRef, age)
	}

	return w.Flush()
}
