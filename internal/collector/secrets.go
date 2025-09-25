package collector

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Secret struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Type        string            `json:"type"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"created_at"`
	DataKeys    []string          `json:"data_keys"`
	Data        map[string]string `json:"data"`
}

func (c *Collector) CollectSecrets(ctx context.Context, namespace string) ([]Secret, error) {
	c.logger.Info("Collecting secrets", "namespace", namespace)

	secretList, err := c.clients.Kubernetes.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	secrets := make([]Secret, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		var dataKeys []string
		dataMap := make(map[string]string)
		for key, value := range secret.Data {
			dataKeys = append(dataKeys, key)
			dataMap[key] = string(value)
		}
		secrets = append(secrets, Secret{
			Name:        secret.Name,
			Namespace:   secret.Namespace,
			Type:        string(secret.Type),
			Labels:      secret.Labels,
			Annotations: AnnotationsCleaner(secret.Annotations),
			CreatedAt:   secret.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			DataKeys:    dataKeys,
			Data:        dataMap,
		})
	}

	c.logger.Info("Successfully collected secrets", "count", len(secrets))
	return secrets, nil
}

type SecretsHandler struct {
	*BaseHandler
}

func NewSecretsHandler() *SecretsHandler {
	return &SecretsHandler{
		BaseHandler: &BaseHandler{
			name:          "secrets",
			clusterScoped: false,
		},
	}
}

func (h *SecretsHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	secrets, err := c.CollectSecrets(ctx, namespace)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(secrets))

	for _, secret := range secrets {
		batch = append(batch, Resource{
			Type:      "secret",
			Namespace: namespace,
			Resource:  secret,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
