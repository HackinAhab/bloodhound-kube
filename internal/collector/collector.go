package collector

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"bloodhound-kube/internal/k8s"
	"bloodhound-kube/internal/logger"
)

type Collector struct {
	clients *k8s.Clients
	logger  *logger.Logger
}

func New(cfg k8s.ClientConfig, log *logger.Logger) (*Collector, error) {
	clients, err := k8s.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clients: %w", err)
	}

	return &Collector{
		clients: clients,
		logger:  log,
	}, nil
}

func (c *Collector) ListNamespaces(ctx context.Context) ([]string, error) {
	c.logger.Info("Listing all namespaces")

	namespaceList, err := c.clients.Kubernetes.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
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
