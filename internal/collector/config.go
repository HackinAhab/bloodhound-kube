package collector

import (
	"bloodhound-kube/internal/utils"
	"fmt"
	"slices"
	"strings"
)

type FetchMode string

const (
	FetchModeFull     FetchMode = "full"
	FetchModeMetadata FetchMode = "metadata"
)

// CollectionsConfig represents the root configuration for resource collection
type CollectionsConfig struct {
	Collections []ResourceCollection
}

// ResourceCollection defines how to collect a specific resource type
type ResourceCollection struct {
	Name              string
	ResourceType      string
	Kind              string
	ShortNames        []string
	APIPath           string
	Description       string
	APIVersion        string
	APIGroup          string
	Plural            string
	Namespaced        bool
	ClusterScoped     bool
	Enabled           bool
	FetchMode         FetchMode
	SupportedClusters []utils.ClusterType
	Custom            bool // Flag for CRDs
}

// Validate checks if the collections configuration is valid
func (c *CollectionsConfig) Validate() error {
	// Validate collections
	if len(c.Collections) == 0 {
		return fmt.Errorf("at least one collection must be defined")
	}

	// Track names and resource types for duplicate detection
	names := make(map[string]bool)
	resourceTypes := make(map[string]bool)

	for i, collection := range c.Collections {
		if err := collection.Validate(); err != nil {
			return fmt.Errorf("collection[%d] (%s) validation failed: %w", i, collection.Name, err)
		}

		// Check for duplicate names
		if names[collection.Name] {
			return fmt.Errorf("duplicate collection name: %s", collection.Name)
		}
		names[collection.Name] = true

		// Check for duplicate resource types
		if resourceTypes[collection.ResourceType] {
			return fmt.Errorf("duplicate resource type: %s (in collection: %s)", collection.ResourceType, collection.Name)
		}
		resourceTypes[collection.ResourceType] = true
	}

	return nil
}

// SetDefaults sets default values for the configuration
func (c *CollectionsConfig) SetDefaults() {
	// Set defaults for each collection
	for i := range c.Collections {
		c.Collections[i].SetDefaults()
	}
}

// Validate checks if a resource collection is valid
func (rc *ResourceCollection) Validate() error {
	// Required fields
	if rc.Name == "" {
		return fmt.Errorf("name is required")
	}
	if rc.ResourceType == "" {
		return fmt.Errorf("resource_type is required")
	}
	if rc.APIVersion == "" {
		return fmt.Errorf("api_version is required")
	}
	if rc.Plural == "" {
		return fmt.Errorf("plural is required")
	}

	switch rc.FetchMode {
	case "", FetchModeFull, FetchModeMetadata:
		// Valid modes
	default:
		return fmt.Errorf("invalid fetch_mode: %s (valid: full, metadata)", rc.FetchMode)
	}

	// Validate name format (lowercase, alphanumeric, hyphens)
	if !isValidName(rc.Name) {
		return fmt.Errorf("invalid name format: %s (must be lowercase alphanumeric with hyphens)", rc.Name)
	}

	// Validate resource type format
	if !isValidResourceType(rc.ResourceType) {
		return fmt.Errorf("invalid resource_type format: %s (must be lowercase alphanumeric with underscores)", rc.ResourceType)
	}

	// Validate supported clusters
	if len(rc.SupportedClusters) == 0 {
		return fmt.Errorf("at least one supported cluster type must be specified")
	}

	for _, cluster := range rc.SupportedClusters {
		if cluster != utils.ClusterTypeKubernetes && cluster != utils.ClusterTypeOpenShift {
			return fmt.Errorf("invalid cluster type: %s (valid: kubernetes, openshift)", cluster)
		}
	}

	// Validate namespaced vs cluster_scoped consistency
	if rc.Namespaced && rc.ClusterScoped {
		return fmt.Errorf("resource cannot be both namespaced and cluster_scoped")
	}

	return nil
}

// SetDefaults sets default values for a resource collection
func (rc *ResourceCollection) SetDefaults() {
	// Default: enabled
	// (We don't set this here because the zero value for bool is false,
	// and we want to allow explicit false values. The loader should handle this.)

	if rc.FetchMode == "" {
		rc.FetchMode = FetchModeFull
	}

	// If neither namespaced nor cluster_scoped is specified, default based on common patterns
	if !rc.Namespaced && !rc.ClusterScoped {
		// Common cluster-scoped resources
		clusterScopedResources := []string{"nodes", "persistentvolumes", "clusterroles", "clusterrolebindings", "crds", "namespaces"}
		nameLower := strings.ToLower(rc.Name)
		for _, csr := range clusterScopedResources {
			if strings.Contains(nameLower, csr) {
				rc.ClusterScoped = true
				return
			}
		}
		// Default to namespaced
		rc.Namespaced = true
	}
}

// GetCollectionsForCluster returns collections supported by the given cluster type
func (c *CollectionsConfig) GetCollectionsForCluster(clusterType utils.ClusterType) []ResourceCollection {
	var supported []ResourceCollection
	for _, collection := range c.Collections {
		if collection.SupportsCluster(clusterType) {
			supported = append(supported, collection)
		}
	}
	return supported
}

// SupportsCluster checks if this collection supports the given cluster type
func (rc *ResourceCollection) SupportsCluster(clusterType utils.ClusterType) bool {
	return slices.Contains(rc.SupportedClusters, clusterType)
}

// isValidName checks if a name is valid (lowercase, alphanumeric, hyphens, underscores)
func isValidName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// isValidResourceType checks if a resource type is valid (lowercase, alphanumeric, underscores)
func isValidResourceType(resourceType string) bool {
	if resourceType == "" {
		return false
	}
	for _, c := range resourceType {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}
