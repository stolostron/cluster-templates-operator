package helm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ghodss/yaml"
	"helm.sh/helm/v3/pkg/repo"
)

func GetIndexFile(indexURL string) (*repo.IndexFile, error) {
	indexFile := &repo.IndexFile{}

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
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
