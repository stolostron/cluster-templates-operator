package helm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"

	clustertemplatev1alpha1 "github.com/rawagner/cluster-templates-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openshiftAPI "github.com/openshift/api/helm/v1beta1"
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

func (h *HelmClient) InstallChart(
	ctx context.Context,
	k8sClient client.Client,
	helmRepository openshiftAPI.HelmChartRepository,
	clusterTemplate clustertemplatev1alpha1.ClusterTemplate,
	clusterTemplateInstance clustertemplatev1alpha1.ClusterTemplateInstance,
) error {

	tlsConfig, err := GetTLSClientConfig(ctx, k8sClient, helmRepository.Spec.ConnectionConfig)

	if err != nil {
		return err
	}

	chartURL, err := GetChartURL(
		tlsConfig,
		helmRepository.Spec.ConnectionConfig.URL,
		clusterTemplate.Spec.HelmChartRef.Name,
		clusterTemplate.Spec.HelmChartRef.Version,
	)

	if err != nil {
		return err
	}

	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}
	httpClient := &http.Client{Transport: tr}

	resp, err := httpClient.Get(chartURL)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Response for %v returned %v with status code %v", chartURL, resp, resp.StatusCode))
	}
	defer resp.Body.Close()

	f, createErr := os.CreateTemp("", "chart-*")
	if createErr != nil {
		return createErr
	}

	defer f.Close()
	defer os.Remove(f.Name())

	_, err = io.Copy(f, resp.Body)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	ch, err := loader.Load(f.Name())
	if err != nil {
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

	cmd := action.NewInstall(h.actionConfig)
	releaseName, _, err := cmd.NameAndChart([]string{clusterTemplateInstance.Name, chartURL})
	if err != nil {
		return err
	}
	cmd.ReleaseName = releaseName
	cmd.Namespace = clusterTemplateInstance.Namespace

	values := make(map[string]interface{})
	err = json.Unmarshal(clusterTemplateInstance.Spec.Values, &values)

	if err != nil {
		return err
	}

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
