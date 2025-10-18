package collector

import (
	"context"
)

// OpenShift-specific collectors
func collectRoutes(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting routes", "namespace", namespace)
	c.logger.Debug("Route collection not yet implemented (requires OpenShift client)", "namespace", namespace)
	// Note: OpenShift routes require the OpenShift client
	// For now, return empty list - can be implemented when OpenShift client is available
	return []any{}, nil
}

func collectProjects(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting projects")
	c.logger.Debug("Project collection not yet implemented (requires OpenShift client)")
	// Note: OpenShift projects require the OpenShift client
	// For now, return empty list - can be implemented when OpenShift client is available
	return []any{}, nil
}

func collectImages(ctx context.Context, c *Collector, namespace string) ([]any, error) {
	c.logger.Info("Collecting images")
	c.logger.Debug("Image collection not yet implemented (requires OpenShift client)")
	// Note: OpenShift images require the OpenShift client
	// For now, return empty list - can be implemented when OpenShift client is available
	return []any{}, nil
}
