package collector

import (
	"bloodhound-kube/internal/utils"
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/metadata"
)

const defaultListPageSize = 500

// CollectorFactory creates collectors from configuration
type CollectorFactory struct {
	clients       *utils.Clients
	logger        utils.Logger
	config        *CollectionsConfig
	dynamicClient dynamic.Interface
	metadata      metadata.Interface
}

// NewCollectorFactory creates a new collector factory
func NewCollectorFactory(clients *utils.Clients, logger utils.Logger, collectionsConfig *CollectionsConfig, dynamicClient dynamic.Interface) (*CollectorFactory, error) {
	if collectionsConfig == nil {
		return nil, fmt.Errorf("collections config is required")
	}

	if dynamicClient == nil {
		return nil, fmt.Errorf("dynamic client is required")
	}

	if clients.Metadata == nil {
		return nil, fmt.Errorf("metadata client is required")
	}

	return &CollectorFactory{
		clients:       clients,
		logger:        logger,
		config:        collectionsConfig,
		dynamicClient: dynamicClient,
		metadata:      clients.Metadata,
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
		kind:          collection.Kind,
		description:   collection.Description,
		clusterScoped: collection.ClusterScoped,
		apiVersion:    collection.APIVersion,
		apiGroup:      collection.APIGroup,
		plural:        collection.Plural,
		fetchMode:     collection.FetchMode,
		dynamicClient: f.dynamicClient,
		metadata:      f.metadata,
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
	kind          string
	description   string
	clusterScoped bool
	apiVersion    string
	apiGroup      string
	plural        string
	fetchMode     FetchMode
	dynamicClient dynamic.Interface
	metadata      metadata.Interface
	logger        utils.Logger
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
func (g *GenericCollector) Collect(ctx context.Context, c *Collector, namespace string) ([]map[string]any, error) {
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

	apiVersion := version
	if g.apiGroup != "" {
		apiVersion = g.apiGroup + "/" + version
	}

	var resources []map[string]any
	continueToken := ""
	page := 0

	for {
		paginateLimit := defaultListPageSize
		if c != nil {
			paginateLimit = c.GetPaginateLimit(defaultListPageSize)
		}
		listOpts := metav1.ListOptions{
			Limit:    int64(paginateLimit),
			Continue: continueToken,
		}

		listStart := time.Now()
		if g.fetchMode == FetchModeMetadata {
			list, listErr := g.listMetadata(ctx, gvr, namespace, listOpts)
			if listErr != nil {
				if g.clusterScoped {
					return nil, fmt.Errorf("failed to list %s metadata: %w", g.plural, listErr)
				}
				return nil, fmt.Errorf("failed to list %s metadata in namespace %s: %w", g.plural, namespace, listErr)
			}
			listDuration := time.Since(listStart)
			page++
			hasContinue := list.GetContinue() != ""
			g.logger.Trace("Listed resources page", "type", g.resourceType, "namespace", namespace, "page", page, "count", len(list.Items), "duration", listDuration, "has_continue", hasContinue, "mode", "metadata")

			processStart := time.Now()
			if resources == nil {
				resources = make([]map[string]any, 0, len(list.Items))
			}
			resources = append(resources, g.buildMetadataResources(list, apiVersion)...)
			g.logger.Trace("Processed resources page", "type", g.resourceType, "namespace", namespace, "page", page, "count", len(list.Items), "duration", time.Since(processStart), "mode", "metadata")

			continueToken = list.GetContinue()
		} else {
			list, listErr := g.listDynamic(ctx, gvr, namespace, listOpts)
			if listErr != nil {
				if g.clusterScoped {
					return nil, fmt.Errorf("failed to list %s: %w", g.plural, listErr)
				}
				return nil, fmt.Errorf("failed to list %s in namespace %s: %w", g.plural, namespace, listErr)
			}
			listDuration := time.Since(listStart)
			page++
			hasContinue := list.GetContinue() != ""
			g.logger.Trace("Listed resources page", "type", g.resourceType, "namespace", namespace, "page", page, "count", len(list.Items), "duration", listDuration, "has_continue", hasContinue, "mode", "full")

			processStart := time.Now()
			if resources == nil {
				resources = make([]map[string]any, 0, len(list.Items))
			}
			redacted := false
			if c != nil {
				redacted = c.IsRedacted()
			}
			resources = append(resources, g.buildDynamicResources(list, namespace, redacted)...)
			g.logger.Trace("Processed resources page", "type", g.resourceType, "namespace", namespace, "page", page, "count", len(list.Items), "duration", time.Since(processStart), "mode", "full")

			continueToken = list.GetContinue()
		}

		if continueToken == "" {
			break
		}
	}

	if g.clusterScoped {
		g.logger.Debug("Collected cluster-scoped resources", "type", g.resourceType, "count", len(resources))
		return resources, nil
	}

	g.logger.Debug("Collected namespaced resources", "type", g.resourceType, "namespace", namespace, "count", len(resources))
	return resources, nil
}

func (g *GenericCollector) listDynamic(ctx context.Context, gvr schema.GroupVersionResource, namespace string, listOpts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	if g.clusterScoped {
		return g.dynamicClient.Resource(gvr).List(ctx, listOpts)
	}
	return g.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, listOpts)
}

func (g *GenericCollector) listMetadata(ctx context.Context, gvr schema.GroupVersionResource, namespace string, listOpts metav1.ListOptions) (*metav1.PartialObjectMetadataList, error) {
	if g.clusterScoped {
		return g.metadata.Resource(gvr).List(ctx, listOpts)
	}
	return g.metadata.Resource(gvr).Namespace(namespace).List(ctx, listOpts)
}

func (g *GenericCollector) buildDynamicResources(list *unstructured.UnstructuredList, namespace string, redacted bool) []map[string]any {
	resources := make([]map[string]any, 0, len(list.Items))
	for _, item := range list.Items {
		processed := applyCollectionHelpers(item.Object, g.plural, redacted)
		if processed == nil {
			continue
		}
		resources = append(resources, processed)
	}
	return resources
}

func (g *GenericCollector) buildMetadataResources(list *metav1.PartialObjectMetadataList, apiVersion string) []map[string]any {
	resources := make([]map[string]any, 0, len(list.Items))
	for _, item := range list.Items {
		kind := g.kind
		if kind == "" {
			kind = item.Kind
		}
		resource := map[string]any{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]any{
				"name":            item.Name,
				"namespace":       item.Namespace,
				"uid":             string(item.UID),
				"resourceVersion": item.ResourceVersion,
				"labels":          mapStringToAny(item.Labels),
				"annotations":     mapStringToAny(item.Annotations),
			},
		}
		resources = append(resources, resource)
	}
	return resources
}

func mapStringToAny(input map[string]string) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
