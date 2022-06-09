package helm

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

var settings = initSettings()

func initSettings() *cli.EnvSettings {
	conf := cli.New()
	conf.RepositoryCache = "/tmp"
	return conf
}

type HelmClient struct {
	*genericclioptions.ConfigFlags
	config       *rest.Config
	actionConfig *action.Configuration
}

func (h HelmClient) ToRESTConfig() (*rest.Config, error) {
	return h.config, nil
}

func NewHelmClient(config *rest.Config) *HelmClient {
	ns := "default"
	helmClient := HelmClient{
		ConfigFlags: &genericclioptions.ConfigFlags{
			APIServer:   &config.Host,
			BearerToken: &config.BearerToken,
			Namespace:   &ns,
			CAFile:      &config.CAFile,
		},
		config: config,
	}

	/*
		inClusterCfg, err := rest.InClusterConfig()

		if err != nil {
			klog.V(4).Info("Running outside cluster, CAFile is unset")
		} else {
			helmClient.ConfigFlags.CAFile = &inClusterCfg.CAFile
		}
	*/

	actionConfig := new(action.Configuration)
	err := actionConfig.Init(helmClient, ns, "secrets", klog.Infof)

	helmClient.actionConfig = actionConfig

	if err != nil {
		fmt.Println(err)
	}

	//repo.NewIndexFile()
	return &helmClient
}
