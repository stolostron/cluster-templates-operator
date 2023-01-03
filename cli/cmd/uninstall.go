package cmd

import (
	"context"
	"fmt"
	"reflect"

	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/cobra"
	res "github.com/stolostron/cluster-templates-operator/cli/installresources"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UninstallOperatorOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
}

func NewUninstallOperatorOptions(streams genericclioptions.IOStreams) *UninstallOperatorOptions {
	return &UninstallOperatorOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdUninstallOperator(
	k8sClient client.Client,
	streams genericclioptions.IOStreams,
) *cobra.Command {
	o := NewUninstallOperatorOptions(streams)
	cmd := &cobra.Command{
		Use:          "uninstall",
		Short:        "Uninstall CLaaS operator",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			//token, err := c.Flags().GetString("token")
			//if err != nil {
			//	return err
			//}
			if err := o.run(k8sClient, ""); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func (o *UninstallOperatorOptions) run(k8sClient client.Client, flag string) error {
	for i := len(res.InstallResources) - 1; i >= 0; i-- {
		res := res.InstallResources[i]
		_, isNs := res.(*corev1.Namespace)
		if !isNs {
			sub, isSubscription := res.(*olm.Subscription)
			if isSubscription {
				subscription := &olm.Subscription{}
				if err := k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(sub), subscription); err != nil {
					fmt.Println("Error getting CSV for subscription " + sub.Name)
					fmt.Println(err.Error())
				} else {
					csv := &olm.ClusterServiceVersion{
						ObjectMeta: v1.ObjectMeta{
							Name:      subscription.Status.CurrentCSV,
							Namespace: subscription.Namespace,
						},
					}
					deleteResource(k8sClient, csv)
				}
			}
			deleteResource(k8sClient, res)
		}
	}
	return nil
}

func deleteResource(k8sClient client.Client, res client.Object) {
	resourceID := reflect.TypeOf(res).String() + "/" + res.GetName()
	if err := k8sClient.Delete(context.TODO(), res); err != nil {
		if !apierrors.IsNotFound(err) {
			fmt.Println("Err removing resource: " + resourceID)
			fmt.Println(err.Error())
		} else {
			fmt.Println("Resource not found: " + resourceID)
		}
	} else {
		fmt.Println("Resource removed: " + resourceID)
	}
}
