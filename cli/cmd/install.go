package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	argoOperator "github.com/argoproj-labs/argocd-operator/api/v1alpha1"
	spin "github.com/briandowns/spinner"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/cobra"
	mce "github.com/stolostron/backplane-operator/api/v1"
	res "github.com/stolostron/cluster-templates-operator/cli/installresources"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	addonapi "open-cluster-management.io/api/addon/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pollingTimeout = 15 * time.Second
)

type InstallOperatorOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
	K8sClient client.Client
	Spinner   *spin.Spinner
}

func NewInstallOperatorOptions(
	streams genericclioptions.IOStreams,
	k8sClient client.Client,
) *InstallOperatorOptions {
	return &InstallOperatorOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
		K8sClient:   k8sClient,
		Spinner:     spin.New(spin.CharSets[14], 100*time.Millisecond, spin.WithWriter(os.Stdout)),
	}
}

func NewCmdInstallOperator(
	k8sClient client.Client,
	streams genericclioptions.IOStreams,
) *cobra.Command {
	o := NewInstallOperatorOptions(streams, k8sClient)
	cmd := &cobra.Command{
		Use:          "install",
		Short:        "Install CLaaS operator, ArgoCD, MCE, and Hypershift",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			//token, err := c.Flags().GetString("token")
			//if err != nil {
			//	return err
			//}
			if err := o.run(""); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func (i *InstallOperatorOptions) run(token string) error {
	ctx := context.TODO()

	fmt.Println("Create CLaaS operator namespace")
	if err := i.createResource(ctx, res.OperatorNs); err != nil {
		return err
	}

	fmt.Println("Create CLaaS operator group")
	if err := i.createResource(ctx, res.OperatorGroup); err != nil {
		return err
	}

	fmt.Println("Create CLaaS operator subscription")
	if err := i.createSubscription(ctx, res.OperatorSubscription, "CLaaS"); err != nil {
		return err
	}
	fmt.Println("CLaaS operator ready")

	fmt.Println("Create ArgoCD namespace")
	if err := i.createResource(ctx, res.ArgoNs); err != nil {
		return err
	}

	fmt.Println("Create ArgoCD sample instance")
	if err := i.createArgoCDInstance(ctx); err != nil {
		return err
	}

	fmt.Println("Create clusters namespace")
	if err := i.createResource(ctx, res.ClusterNs); err != nil {
		return err
	}

	fmt.Println("Create pull-secret secret")
	if err := i.K8sClient.Create(ctx, res.PullSecret); err != nil {
		return err
	}

	fmt.Println("Create sshkey secret")
	if err := i.K8sClient.Create(ctx, res.SshKeySecret); err != nil {
		return err
	}

	fmt.Println("Create MCE operator namespace")
	if err := i.createResource(ctx, res.MceNs); err != nil {
		return err
	}

	fmt.Println("Create MCE operator group")
	if err := i.createResource(ctx, res.MceOperatorGroup); err != nil {
		return err
	}

	fmt.Println("Create MCE operator subscription")
	if err := i.createSubscription(ctx, res.MceSub, "MCE"); err != nil {
		return err
	}
	fmt.Println("MCE operator ready")

	time.Sleep(2 * time.Minute)
	fmt.Println("Create MCE operator config")
	if err := i.createMCEcr(ctx); err != nil {
		return err
	}

	fmt.Println("Create Managed cluster")
	if err := i.createResource(ctx, res.MceManagedCluster); err != nil {
		return err
	}

	fmt.Println("Enable Hypershift")
	if err := i.createHypershiftAddon(ctx); err != nil {
		return err
	}

	fmt.Println("Environment ready!")
	return nil
}

func (i *InstallOperatorOptions) createResource(ctx context.Context, resource client.Object) error {
	if err := i.K8sClient.Create(ctx, resource); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func (i *InstallOperatorOptions) createSubscription(
	ctx context.Context,
	resource client.Object,
	operatorName string,
) error {
	if err := i.createResource(ctx, resource); err != nil {
		return err
	}

	sub := &olm.Subscription{}
	i.Spinner.Start()
	for {
		i.Spinner.Suffix = " waiting for " + operatorName + " operator subscription"
		if err := i.K8sClient.Get(ctx, client.ObjectKeyFromObject(resource), sub); err != nil {
			i.Spinner.Suffix = " error retrieving " + operatorName + " operator subscription: " + err.Error()
		}

		ready := false
		for _, condition := range sub.Status.Conditions {
			if condition.Type == "CatalogSourcesUnhealthy" {
				if condition.Status == corev1.ConditionFalse &&
					condition.Reason == "AllCatalogSourcesHealthy" &&
					sub.Status.CurrentCSV != "" {
					ready = true
				}
			}
		}
		if ready {
			break
		}
		time.Sleep(pollingTimeout)
	}
	i.Spinner.Stop()

	csv := &olm.ClusterServiceVersion{}
	i.Spinner.Start()
	for {
		i.Spinner.Suffix = " waiting for " + operatorName + " operator"
		if err := i.K8sClient.Get(ctx, client.ObjectKey{Namespace: client.ObjectKeyFromObject(resource).Namespace, Name: sub.Status.CurrentCSV}, csv); err != nil {
			i.Spinner.Suffix = " error retrieving " + operatorName + " operator CSV: " + err.Error()
		}

		if csv.Status.Phase == "Succeeded" {
			break
		}
		time.Sleep(pollingTimeout)
	}
	i.Spinner.Stop()

	return nil
}

func (i *InstallOperatorOptions) createArgoCDInstance(ctx context.Context) error {
	if err := i.createResource(ctx, res.ArgoInstance); err != nil {
		return err
	}
	argoInstance := &argoOperator.ArgoCD{}
	i.Spinner.Start()
	for {
		if err := i.K8sClient.Get(ctx, client.ObjectKeyFromObject(res.ArgoInstance), argoInstance); err != nil {
			i.Spinner.Suffix = " error retrieving ArgoCD sample instance"
		}

		if argoInstance.Status.Phase != "Available" {
			i.Spinner.Suffix = " waiting for ArgoCD instance"
		} else if argoInstance.Status.Server != "Running" {
			i.Spinner.Suffix = " waiting for ArgoCD instance server"
		} else if argoInstance.Status.Repo != "Running" {
			i.Spinner.Suffix = " waiting for ArgoCD instance repo"
		} else if argoInstance.Status.ApplicationController != "Running" {
			i.Spinner.Suffix = " waiting for ArgoCD instance controller"
			//} else if argoInstance.Status.Redis != "Running" {
			//	s.Suffix = " waiting for ArgoCD instance redis"
		} else {
			break
		}
		time.Sleep(pollingTimeout)
	}
	i.Spinner.Stop()
	return nil
}

func (i *InstallOperatorOptions) createMCEcr(ctx context.Context) error {
	if err := i.createResource(ctx, res.Mce); err != nil {
		return err
	}

	mceResource := &mce.MultiClusterEngine{}

	i.Spinner.Start()
	for {
		if err := i.K8sClient.Get(ctx, client.ObjectKeyFromObject(res.Mce), mceResource); err != nil {
			fmt.Println("error retrieving MCE")
		}

		if mceResource.Status.Phase == "Available" {
			break
		}

		i.Spinner.Suffix = " waiting for MCE"
		time.Sleep(pollingTimeout)
	}
	i.Spinner.Stop()
	return nil
}

func (i *InstallOperatorOptions) createHypershiftAddon(ctx context.Context) error {
	if err := i.createResource(ctx, res.MceHypershiftAddon); err != nil {
		return err
	}

	hypershiftAddon := &addonapi.ManagedClusterAddOn{}

	i.Spinner.Start()
	for {
		if err := i.K8sClient.Get(ctx, client.ObjectKeyFromObject(res.MceHypershiftAddon), hypershiftAddon); err != nil {
			fmt.Println("error retrieving Hypershift AddOn")
		}

		ready := false
		for _, condition := range hypershiftAddon.Status.Conditions {
			if condition.Type == "Available" && condition.Status == "True" {
				ready = true
			}
		}

		if ready {
			break
		}

		i.Spinner.Suffix = " waiting for Hypershift AddOn"
		time.Sleep(pollingTimeout)
	}
	i.Spinner.Stop()
	return nil
}
