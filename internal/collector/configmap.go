package collector

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMap struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	DataKeys       []string          `json:"data_keys"`
	BinaryDataKeys []string          `json:"binary_data_keys,omitempty"`
}

func (c *Collector) CollectConfigMaps(ctx context.Context, namespace string) ([]ConfigMap, error) {
	c.logger.Info("Collecting configmaps", "namespace", namespace)

	configMapList, err := c.clients.Kubernetes.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps: %w", err)
	}

	configMaps := make([]ConfigMap, 0, len(configMapList.Items))
	for _, cm := range configMapList.Items {
		var dataKeys []string
		for key := range cm.Data {
			dataKeys = append(dataKeys, key)
		}

		var binaryDataKeys []string
		for key := range cm.BinaryData {
			binaryDataKeys = append(binaryDataKeys, key)
		}

		configMaps = append(configMaps, ConfigMap{
			Name:           cm.Name,
			Namespace:      cm.Namespace,
			Labels:         cm.Labels,
			Annotations:    cm.Annotations,
			DataKeys:       dataKeys,
			BinaryDataKeys: binaryDataKeys,
		})
	}

	c.logger.Info("Successfully collected configmaps", "count", len(configMaps))
	return configMaps, nil
}

type ConfigMapsHandler struct {
	*BaseHandler
}

func NewConfigMapsHandler() *ConfigMapsHandler {
	return &ConfigMapsHandler{
		BaseHandler: &BaseHandler{
			name:          "configmaps",
			clusterScoped: false,
		},
	}
}

func (h *ConfigMapsHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	configMaps, err := c.CollectConfigMaps(ctx, namespace)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(configMaps))

	for _, configMap := range configMaps {
		batch = append(batch, Resource{
			Type:      "configmap",
			Namespace: namespace,
			Resource:  configMap,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
