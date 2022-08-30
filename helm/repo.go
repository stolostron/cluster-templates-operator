package helm

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ghodss/yaml"
	"helm.sh/helm/v3/pkg/repo"
)

func GetIndexFile(tlsConfig *tls.Config, indexURL string) (*repo.IndexFile, error) {
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
		return nil, errors.New(fmt.Sprintf("Response for %v returned %v with status code %v", indexURL, resp, resp.StatusCode))
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(body, indexFile)
	return indexFile, err
}

func GetChartURL(tlsConfig *tls.Config, indexURL string, chartName string, chartVersion string) (string, error) {
	indexFile, err := GetIndexFile(tlsConfig, indexURL)
	if err != nil {
		return "", err
	}

	var helmChartURL string
	entry := indexFile.Entries[chartName]
	for _, e := range entry {
		if e.Version == chartVersion {
			helmChartURL = e.URLs[0]
			break
		}
	}

	if helmChartURL == "" {
		return "", errors.New("could not find helm chart")
	}

	if strings.HasSuffix(indexURL, "/index.yaml") {
		indexURL = strings.TrimSuffix(indexURL, "index.yaml")
	}

	helmChartURL, err = repo.ResolveReferenceURL(indexURL, helmChartURL)
	if err != nil {
		return "", errors.New("error resolving chart url")
	}
	return helmChartURL, nil
}
