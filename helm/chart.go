package helm

import (
	"context"
	"fmt"
	"io"
	"os"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *HelmClient) GetChart(
	ctx context.Context,
	k8sClient client.Client,
	repoURL string,
	chartName string,
	version string,
	argoCDNamespace string,
) (*chart.Chart, error) {

	secret, err := GetRepoSecret(ctx, k8sClient, argoCDNamespace, repoURL)
	if err != nil {
		return nil, err
	}
	cm, err := GetRepoCM(ctx, k8sClient, argoCDNamespace)
	if err != nil {
		return nil, err
	}
	httpClient, err := GetRepoHTTPClient(ctx, repoURL, secret, cm)

	if err != nil {
		return nil, err
	}
	chartURL, err := getChartURL(
		httpClient,
		repoURL,
		chartName,
		version,
		secret,
	)

	if err != nil {
		return nil, err
	}

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
