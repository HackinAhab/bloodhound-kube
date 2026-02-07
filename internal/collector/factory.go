package collector

import (
	"bloodhound-kube/internal/utils"
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// CollectorFactory creates collectors from configuration
type CollectorFactory struct {
	clients       *utils.Clients
	logger        *utils.Logger
	config        *CollectionsConfig
	dynamicClient dynamic.Interface
}

// NewCollectorFactory creates a new collector factory
func NewCollectorFactory(clients *utils.Clients, logger *utils.Logger, collectionsConfig *CollectionsConfig, dynamicClient dynamic.Interface) (*CollectorFactory, error) {
	if collectionsConfig == nil {
		return nil, fmt.Errorf("collections config is required")
	}

	if dynamicClient == nil {
		return nil, fmt.Errorf("dynamic client is required")
	}

	return &CollectorFactory{
		clients:       clients,
		logger:        logger,
		config:        collectionsConfig,
		dynamicClient: dynamicClient,
	}, nil
}

// CreateCollector creates a collector for a specific resource collection
func (f *CollectorFactory) CreateCollector(collection ResourceCollection) (ResourceHandler, error) {
	// Validate that the resource is supported by the current cluster type
	clusterType := f.clients.ClusterType
	if !collection.SupportsCluster(clusterType) {
		return nil, fmt.Errorf("resource %s not supported on cluster type %s", collection.Name, clusterType)
	}

	// Create generic collector
	return &GenericCollector{
		name:          collection.Name,
		resourceType:  collection.ResourceType,
		description:   collection.Description,
		clusterScoped: collection.ClusterScoped,
		apiVersion:    collection.APIVersion,
		apiGroup:      collection.APIGroup,
		plural:        collection.Plural,
		dynamicClient: f.dynamicClient,
		logger:        f.logger,
		rateLimit:     collection.RateLimit,
	}, nil
}

// CreateAllCollectors creates collectors for all enabled collections
func (f *CollectorFactory) CreateAllCollectors() (map[string]ResourceHandler, error) {
	handlers := make(map[string]ResourceHandler)

	// Get collections that are enabled and supported by this cluster
	clusterType := f.clients.ClusterType
	collections := f.config.GetCollectionsForCluster(clusterType)

	for _, collection := range collections {
		if !collection.Enabled {
			continue
		}

		handler, err := f.CreateCollector(collection)
		if err != nil {
			f.logger.Warn("Failed to create collector", "resource", collection.Name, "error", err)
			continue
		}

		// Register handler by name
		handlers[collection.Name] = handler

		// Also register by nicknames
		for _, nickname := range collection.Nicknames {
			handlers[nickname] = handler
		}
	}

	f.logger.Info("Created collectors from config", "count", len(handlers))
	return handlers, nil
}

// GetCollectionByName returns a collection definition by name or nickname
func (f *CollectorFactory) GetCollectionByName(name string) *ResourceCollection {
	return f.config.GetByName(name)
}

// GetCollectionByResourceType returns a collection definition by resource type
func (f *CollectorFactory) GetCollectionByResourceType(resourceType string) *ResourceCollection {
	return f.config.GetByResourceType(resourceType)
}

// ShouldCollectNamespace checks if a namespace should be collected based on config
func (f *CollectorFactory) ShouldCollectNamespace(namespace string) bool {
	return f.config.Namespaces.ShouldCollectNamespace(namespace)
}

// GetPerformanceSettings returns the performance settings from config
func (f *CollectorFactory) GetPerformanceSettings() PerformanceSettings {
	return f.config.Settings
}

// extractVersion extracts the version part from an API version string
// Examples:
//   - "v1" -> "v1"
//   - "apps/v1" -> "v1"
//   - "rbac.authorization.k8s.io/v1" -> "v1"
func extractVersion(apiVersion string) string {
	// If there's a slash, take the part after it
	for i := len(apiVersion) - 1; i >= 0; i-- {
		if apiVersion[i] == '/' {
			return apiVersion[i+1:]
		}
	}
	// No slash found, return as-is
	return apiVersion
}

// GenericCollector is a dynamic collector that works with any Kubernetes resource
type GenericCollector struct {
	name          string
	resourceType  string
	description   string
	clusterScoped bool
	apiVersion    string
	apiGroup      string
	plural        string
	dynamicClient dynamic.Interface
	logger        *utils.Logger
	rateLimit     int
}

// GetName returns the collector name
func (g *GenericCollector) GetName() string {
	return g.name
}

// IsClusterScoped returns whether the resource is cluster-scoped
func (g *GenericCollector) IsClusterScoped() bool {
	return g.clusterScoped
}

// GetDescription returns the collector description
func (g *GenericCollector) GetDescription() string {
	return g.description
}

// GetSupportedClusterTypes returns supported cluster types (generic supports all)
func (g *GenericCollector) GetSupportedClusterTypes() []utils.ClusterType {
	return []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift}
}

// Collect collects resources using the dynamic client
func (g *GenericCollector) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	// Parse API version to extract just the version part
	// API versions can be in format "v1" or "apps/v1"
	version := g.apiVersion
	if g.apiGroup != "" {
		// If we have an API group, the version should not include the group
		// Extract version from "group/version" format if needed
		version = extractVersion(g.apiVersion)
	}

	// Create GroupVersionResource
	gvr := schema.GroupVersionResource{
		Group:    g.apiGroup,
		Version:  version,
		Resource: g.plural,
	}

	// List options
	listOpts := metav1.ListOptions{}

	if g.clusterScoped {
		// Cluster-scoped resource
		unstructuredList, err := g.dynamicClient.Resource(gvr).List(ctx, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list %s: %w", g.plural, err)
		}

		// Convert to resources
		resources := make([]Resource, len(unstructuredList.Items))
		for i, item := range unstructuredList.Items {
			processed := ApplyCollectionHelpers(item.Object, g.plural, c.IsRedacted())
			resources[i] = Resource{
				Type:      g.resourceType,
				Namespace: namespace,
				Resource:  processed,
				Timestamp: metav1.Now().Format("2006-01-02T15:04:05Z07:00"),
			}
		}

		g.logger.Debug("Collected cluster-scoped resources",
			"type", g.resourceType,
			"count", len(resources))

		return resources, nil
	} else {
		// Namespaced resource
		unstructuredList, err := g.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list %s in namespace %s: %w", g.plural, namespace, err)
		}

		// Convert to resources
		resources := make([]Resource, len(unstructuredList.Items))
		for i, item := range unstructuredList.Items {
			processed := ApplyCollectionHelpers(item.Object, g.plural, c.IsRedacted())
			resources[i] = Resource{
				Type:      g.resourceType,
				Namespace: namespace,
				Resource:  processed,
				Timestamp: metav1.Now().Format("2006-01-02T15:04:05Z07:00"),
			}
		}

		g.logger.Debug("Collected namespaced resources",
			"type", g.resourceType,
			"namespace", namespace,
			"count", len(resources))

		return resources, nil
	}
}
