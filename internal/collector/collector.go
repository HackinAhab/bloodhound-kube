package collector

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"bloodhound-kube/internal/k8s"
	"bloodhound-kube/internal/logger"
)

type Collector struct {
	client *kubernetes.Clientset
	logger *logger.Logger
}

func New(log *logger.Logger) (*Collector, error) {
	client, err := k8s.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Collector{
		client: client,
		logger: log,
	}, nil
}

func (c *Collector) ListNamespaces(ctx context.Context) ([]string, error) {
	c.logger.Info("Listing all namespaces")

	namespaceList, err := c.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var namespaces []string
	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, ns.Name)
	}

	c.logger.Info("Successfully listed namespaces", "count", len(namespaces))
	return namespaces, nil
}
