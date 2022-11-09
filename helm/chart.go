package helm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func (h *HelmClient) GetChart(
	ctx context.Context,
	repoURL string,
	chartName string,
	version string,
) (*chart.Chart, error) {

	/*
		tlsConfig, err := h.GetTLSClientConfig(ctx, helmRepository.Spec.ConnectionConfig)

		if err != nil {
			return err
		}
	*/

	chartURL, err := getChartURL(
		nil, //tlsConfig
		repoURL,
		chartName,
		version,
	)

	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		//TLSClientConfig: tlsConfig,
	}}

	resp, err := httpClient.Get(chartURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf(
			"response for %v returned %v with status code %v",
			chartURL,
			resp,
			resp.StatusCode,
		)
	}
	defer resp.Body.Close()

	f, err := os.CreateTemp("", "chart-*")
	if err != nil {
		return nil, err
	}

	defer f.Close()
	defer os.Remove(f.Name())

	_, err = io.Copy(f, resp.Body)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return loader.Load(f.Name())
}
