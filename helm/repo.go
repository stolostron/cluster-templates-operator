package helm

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ghodss/yaml"
	"helm.sh/helm/v3/pkg/repo"
)

func GetIndexFile(httpClient *http.Client, indexURL string) (*repo.IndexFile, error) {
	indexFile := &repo.IndexFile{}

	if !strings.HasSuffix(indexURL, "/index.yaml") {
		indexURL += "/index.yaml"
	}
	resp, err := httpClient.Get(indexURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf(
			"response for %v returned %v with status code %v",
			indexURL,
			resp,
			resp.StatusCode,
		)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(body, indexFile)
	return indexFile, err
}

func getChartURL(
	httpClient *http.Client,
	indexURL string,
	chartName string,
	chartVersion string,
) (string, error) {
	indexFile, err := GetIndexFile(httpClient, indexURL)
	if err != nil {
		return "", err
	}

	helmChartURL := ""
	entry := indexFile.Entries[chartName]
	for _, e := range entry {
		if e.Version == chartVersion {
			helmChartURL = e.URLs[0]
			break
		}
	}

	if helmChartURL == "" {
		return "", fmt.Errorf("could not find helm chart")
	}

	if strings.HasSuffix(indexURL, "/index.yaml") {
		indexURL = strings.TrimSuffix(indexURL, "index.yaml")
	}

	helmChartURL, err = repo.ResolveReferenceURL(indexURL, helmChartURL)
	if err != nil {
		return "", fmt.Errorf("error resolving chart url - %q", err)
	}
	return helmChartURL, nil
}
