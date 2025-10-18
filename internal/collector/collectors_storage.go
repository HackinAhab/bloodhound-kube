package collector

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This file contains collection functions for Kubernetes storage resources:
// ConfigMaps, Secrets, and related collection logic.

// collectConfigMaps collects Kubernetes ConfigMaps from the specified namespace
func collectConfigMaps(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting configmaps", "namespace", namespace)
	c.logger.Debug("Starting configmap collection", "namespace", namespace)

	configMapList, err := c.clients.Kubernetes.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list configmaps", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list configmaps: %w", err)
	}

	c.logger.Debug("Retrieved configmap list", "namespace", namespace, "count", len(configMapList.Items))

	configMaps := make([]any, 0, len(configMapList.Items))
	for _, cm := range configMapList.Items {
		var dataKeys []string
		var dataMap map[string]string
		var binaryDataKeys []string
		var binaryDataMap map[string][]byte

		if c.IsRedacted() {
			// When redacted, collect key names but redact values
			for key := range cm.Data {
				dataKeys = append(dataKeys, key)
			}
			for key := range cm.BinaryData {
				binaryDataKeys = append(binaryDataKeys, key)
			}
			dataMap = nil
			binaryDataMap = nil
		} else {
			// Normal collection - include keys and data
			dataMap = make(map[string]string)
			for key, value := range cm.Data {
				dataKeys = append(dataKeys, key)
				dataMap[key] = value
			}

			binaryDataMap = make(map[string][]byte)
			for key, value := range cm.BinaryData {
				binaryDataKeys = append(binaryDataKeys, key)
				binaryDataMap[key] = value
			}
		}

		configMaps = append(configMaps, ConfigMap{
			CommonResourceMeta: CommonResourceMeta{
				Name:        cm.Name,
				Namespace:   cm.Namespace,
				Labels:      cm.Labels,
				Annotations: AnnotationsCleaner(cm.Annotations),
				CreatedAt:   cm.CreationTimestamp.Time,
			},
			DataKeys:       dataKeys,
			Data:           dataMap,
			BinaryDataKeys: binaryDataKeys,
			BinaryData:     binaryDataMap,
		})
	}

	c.logger.Info("Successfully collected configmaps", "namespace", namespace, "count", len(configMaps))
	c.logger.Debug("Configmap collection completed", "namespace", namespace, "processed", len(configMaps))
	return configMaps, nil
}

// collectSecrets collects Kubernetes Secrets from the specified namespace
func collectSecrets(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting secrets", "namespace", namespace)
	c.logger.Debug("Starting secret collection", "namespace", namespace)

	secretList, err := c.clients.Kubernetes.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list secrets", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	c.logger.Debug("Retrieved secret list", "namespace", namespace, "count", len(secretList.Items))

	secrets := make([]any, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		var dataKeys []string
		var dataMap map[string]string

		if c.IsRedacted() {
			// When redacted, collect key names but redact values
			for key := range secret.Data {
				dataKeys = append(dataKeys, key)
			}
			dataMap = nil
		} else {
			dataMap = make(map[string]string)
			for key, value := range secret.Data {
				dataKeys = append(dataKeys, key)
				dataMap[key] = string(value)
			}
		}

		secrets = append(secrets, Secret{
			CommonResourceMeta: CommonResourceMeta{
				Name:        secret.Name,
				Namespace:   secret.Namespace,
				Labels:      secret.Labels,
				Annotations: AnnotationsCleaner(secret.Annotations),
				CreatedAt:   secret.CreationTimestamp.Time,
			},
			Type:     string(secret.Type),
			DataKeys: dataKeys,
			Data:     dataMap,
		})
	}

	c.logger.Info("Successfully collected secrets", "namespace", namespace, "count", len(secrets))
	c.logger.Debug("Secret collection completed", "namespace", namespace, "processed", len(secrets))
	return secrets, nil
}
