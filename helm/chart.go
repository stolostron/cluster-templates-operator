package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"

	"github.com/rawagner/cluster-templates-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openshiftAPI "github.com/openshift/api/helm/v1beta1"
)

func (h *HelmClient) InstallChart(
	ctx context.Context,
	helmRepository openshiftAPI.HelmChartRepository,
	clusterTemplate v1alpha1.ClusterTemplate,
	clusterTemplateInstance v1alpha1.ClusterTemplateInstance,
) error {

	tlsConfig, err := h.GetTLSClientConfig(ctx, helmRepository.Spec.ConnectionConfig)

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

	httpClient := &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}}

	resp, err := httpClient.Get(chartURL)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf(
			"response for %v returned %v with status code %v",
			chartURL,
			resp,
			resp.StatusCode,
		)
	}
	defer resp.Body.Close()

	f, err := os.CreateTemp("", "chart-*")
	if err != nil {
		return err
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
	cmd.ReleaseName = clusterTemplateInstance.Name
	cmd.Namespace = clusterTemplateInstance.Namespace

	values := make(map[string]interface{})
	if len(clusterTemplateInstance.Spec.Values) > 0 {
		if err := json.Unmarshal(clusterTemplateInstance.Spec.Values, &values); err != nil {
			return err
		}
	}

	templateValues := make(map[string]interface{})
	for _, prop := range clusterTemplate.Spec.Properties {
		// filter out ClusterSetup properties
		if !prop.ClusterSetup {
			if len(prop.DefaultValue) != 0 {
				value := new(interface{})
				if err = json.Unmarshal(prop.DefaultValue, &value); err != nil {
					return err
				}
				templateValues[prop.Name] = &value
			} else if prop.SecretRef != nil {
				valueSecret := corev1.Secret{}

				if err := h.k8sClient.Get(
					ctx,
					client.ObjectKey{
						Name:      prop.SecretRef.Name,
						Namespace: prop.SecretRef.Namespace,
					},
					&valueSecret,
				); err != nil {
					return err
				}
				templateValues[prop.Name] = string(valueSecret.Data[prop.Name])

			} else {
				if val, ok := values[prop.Name]; ok {
					templateValues[prop.Name] = val
				}
			}
		}
	}

	_, err = cmd.Run(ch, templateValues)
	return err
}

func (h *HelmClient) GetRelease(releaseName string) (*release.Release, error) {
	cmd := action.NewGet(h.actionConfig)
	return cmd.Run(releaseName)
}

func (h *HelmClient) UninstallRelease(
	releaseName string,
) (*release.UninstallReleaseResponse, error) {
	cmd := action.NewUninstall(h.actionConfig)
	return cmd.Run(releaseName)
}
