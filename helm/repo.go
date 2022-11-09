package helm

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ghodss/yaml"
	"helm.sh/helm/v3/pkg/repo"
)

func getIndexFile(tlsConfig *tls.Config, indexURL string) (*repo.IndexFile, error) {
	indexFile := &repo.IndexFile{}
	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}
	httpClient := &http.Client{Transport: tr}

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
	tlsConfig *tls.Config,
	indexURL string,
	chartName string,
	chartVersion string,
) (string, error) {
	indexFile, err := getIndexFile(tlsConfig, indexURL)
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
