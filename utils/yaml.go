package utils

import (
	"encoding/json"

	"encoding/base64"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

type TLSClientConfig struct {
	CAData string `json:"caData"`
}

type ClusterConfig struct {
	BearerToken     string          `json:"bearerToken"`
	TLSClientConfig TLSClientConfig `json:"tlsClientConfig"`
}

func GetValueFromSecret(secret corev1.Secret, key string) (string, error) {
	var data string
	err := yaml.Unmarshal(secret.Data[key], &data)
	return data, err
}

func GetMapValueFromSecret(secret corev1.Secret, key string) (ClusterConfig, error) {
	data := ClusterConfig{}
	err := json.Unmarshal(secret.Data[key], &data)

	if err != nil {
		return data, err
	}
	decoded, err := base64.StdEncoding.DecodeString(data.TLSClientConfig.CAData)
	data.TLSClientConfig.CAData = string(decoded)
	return data, err
}
