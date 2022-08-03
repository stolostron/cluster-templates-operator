package helm

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
)

func (h *HelmClient) GetChart(chartURL string) (*chart.Chart, error) {
	cmd := action.NewInstall(h.actionConfig)

	//"https://github.com/openshift-helm-charts/charts/releases/download/redhat-dotnet-0.0.1/redhat-dotnet-0.0.1.tgz"
	chartLocation, err := cmd.ChartPathOptions.LocateChart(chartURL, settings)
	if err != nil {
		return nil, err
	}

	return loader.Load(chartLocation)
}

func (h *HelmClient) InstallChart(chartURL string, releaseName string, releaseNamespace string, values map[string]interface{}) error {
	cmd := action.NewInstall(h.actionConfig)

	releaseName, chartName, err := cmd.NameAndChart([]string{releaseName, chartURL})
	if err != nil {
		fmt.Println("1")
		return err
	}
	cmd.ReleaseName = releaseName

	cp, err := cmd.ChartPathOptions.LocateChart(chartName, settings)
	if err != nil {
		fmt.Println("2")
		return err
	}

	ch, err := loader.Load(cp)
	if err != nil {
		fmt.Println("3")
		return err
	}

	// Add chart URL as an annotation before installation
	if ch.Metadata == nil {
		ch.Metadata = new(chart.Metadata)
	}
	if ch.Metadata.Annotations == nil {
		ch.Metadata.Annotations = make(map[string]string)
	}
	ch.Metadata.Annotations["chart_url"] = chartURL

	cmd.Namespace = releaseNamespace
	_, err = cmd.Run(ch, values)
	if err != nil {
		return err
	}
	return nil
}

func (h *HelmClient) GetRelease(releaseName string) (*release.Release, error) {
	cmd := action.NewGet(h.actionConfig)
	return cmd.Run(releaseName)
}

func (h *HelmClient) UninstallRelease(releaseName string) (*release.UninstallReleaseResponse, error) {
	cmd := action.NewUninstall(h.actionConfig)
	return cmd.Run(releaseName)
}
