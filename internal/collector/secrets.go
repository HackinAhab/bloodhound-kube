package collector

import (
	"context"
	"fmt"

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
}

func (c *Collector) CollectSecrets(ctx context.Context, namespace string) ([]Secret, error) {
	c.logger.Info("Collecting secrets", "namespace", namespace)

	secretList, err := c.client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	secrets := make([]Secret, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		var dataKeys []string
		for key := range secret.Data {
			dataKeys = append(dataKeys, key)
		}

		secrets = append(secrets, Secret{
			Name:        secret.Name,
			Namespace:   secret.Namespace,
			Type:        string(secret.Type),
			Labels:      secret.Labels,
			Annotations: secret.Annotations,
			CreatedAt:   secret.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			DataKeys:    dataKeys,
		})
	}

	c.logger.Info("Successfully collected secrets", "count", len(secrets))
	return secrets, nil
}
