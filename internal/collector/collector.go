package collector

import (
	"bloodhound-kube/internal/utils"
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Collector struct {
	clients       *utils.Clients
	logger        utils.Logger
	redacted      bool
	paginateLimit int
}

func New(cfg utils.ClientConfig, log utils.Logger) (*Collector, error) {
	clients, err := utils.NewClient(cfg)
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
	c.logger.Debug("Starting namespace enumeration")

	namespaceList, err := c.clients.Kubernetes.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to list namespaces", "error", err)
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	c.logger.Debug("Retrieved namespace list from API", "raw_count", len(namespaceList.Items))

	var namespaces []string
	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, ns.Name)
		c.logger.Debug("Found namespace", "name", ns.Name)
	}

	c.logger.Info("Successfully listed namespaces", "count", len(namespaces))
	c.logger.Debug("Namespace enumeration completed", "namespaces", namespaces)
	return namespaces, nil
}

func (c *Collector) IsOpenShift() bool {
	return c.clients.IsOpenShift()
}

func (c *Collector) GetPlatform() string {
	return c.clients.GetPlatform()
}

func (c *Collector) GetClusterType() utils.ClusterType {
	return c.clients.ClusterType
}

func (c *Collector) SetRedacted(redacted bool) {
	c.redacted = redacted
}

func (c *Collector) SetPaginateLimit(limit int) {
	c.paginateLimit = limit
}

func (c *Collector) IsRedacted() bool {
	return c.redacted
}

func (c *Collector) GetPaginateLimit(defaultLimit int) int {
	if c.paginateLimit > 0 {
		return c.paginateLimit
	}
	return defaultLimit
}

// GetClients returns the Kubernetes clients
func (c *Collector) GetClients() *utils.Clients {
	return c.clients
}
