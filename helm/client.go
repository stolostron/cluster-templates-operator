package helm

import (
	"fmt"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func initSettings() *cli.EnvSettings {
	conf := cli.New()
	conf.RepositoryCache = "/tmp"
	return conf
}

type HelmClient struct {
	*genericclioptions.ConfigFlags
	config       *rest.Config
	actionConfig *action.Configuration
	k8sClient    client.Client
}

func NewHelmClient(config *rest.Config, k8sClient client.Client, certDataFileName *string, keyDataFileName *string, caDataFileName *string) *HelmClient {
	initSettings()
	ns := "default"

	configFlags := &genericclioptions.ConfigFlags{
		APIServer:   &config.Host,
		BearerToken: &config.BearerToken,
		Namespace:   &ns,
		CAFile:      &config.CAFile,
	}

	if caDataFileName != nil {
		configFlags.CAFile = caDataFileName
	}
	if certDataFileName != nil {
		configFlags.CertFile = certDataFileName
	}
	if keyDataFileName != nil {
		configFlags.KeyFile = keyDataFileName
	}

	helmClient := HelmClient{
		ConfigFlags: configFlags,
		config:      config,
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(helmClient, ns, "secrets", klog.Infof); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	helmClient.actionConfig = actionConfig
	helmClient.k8sClient = k8sClient

	return &helmClient
}
