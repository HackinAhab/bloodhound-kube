package config

import (
	"fmt"
	"strings"
)

// CollectionsConfig represents the root configuration for resource collection
type CollectionsConfig struct {
	Version     string               `yaml:"version"`
	Metadata    ConfigMetadata       `yaml:"metadata"`
	Settings    PerformanceSettings  `yaml:"settings,omitempty"`
	Namespaces  NamespaceFilter      `yaml:"namespaces,omitempty"`
	Collections []ResourceCollection `yaml:"collections"`
}

// ResourceCollection defines how to collect a specific resource type
type ResourceCollection struct {
	Name              string        `yaml:"name"`
	Nicknames         []string      `yaml:"nicknames,omitempty"`
	ResourceType      string        `yaml:"resource_type"`
	Description       string        `yaml:"description,omitempty"`
	APIVersion        string        `yaml:"api_version"`
	APIGroup          string        `yaml:"api_group"`
	Plural            string        `yaml:"plural"`
	Namespaced        bool          `yaml:"namespaced"`
	ClusterScoped     bool          `yaml:"cluster_scoped"`
	Enabled           bool          `yaml:"enabled"`
	SupportedClusters []ClusterType `yaml:"supported_clusters"`
	Custom            bool          `yaml:"custom,omitempty"`     // Flag for CRDs
	RateLimit         int           `yaml:"rate_limit,omitempty"` // Per-resource rate limit
}

// Validate checks if the collections configuration is valid
func (c *CollectionsConfig) Validate() error {
	// Validate version
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.Version != string(ConfigVersion1_0) {
		return fmt.Errorf("unsupported config version: %s (supported: %s)", c.Version, ConfigVersion1_0)
	}

	// Validate metadata
	if err := c.Metadata.Validate(); err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}

	// Validate settings
	if err := c.Settings.Validate(); err != nil {
		return fmt.Errorf("settings validation failed: %w", err)
	}

	// Validate namespace filter
	if err := c.Namespaces.Validate(); err != nil {
		return fmt.Errorf("namespace filter validation failed: %w", err)
	}

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

		// Check for duplicate nicknames
		for _, nickname := range collection.Nicknames {
			if names[nickname] {
				return fmt.Errorf("duplicate nickname: %s (conflicts with collection: %s)", nickname, collection.Name)
			}
			names[nickname] = true
		}
	}

	return nil
}

// SetDefaults sets default values for the configuration
func (c *CollectionsConfig) SetDefaults() {
	// Set default settings
	c.Settings.SetDefaults()

	// Set default namespace mode if not specified
	if c.Namespaces.Mode == "" {
		c.Namespaces.Mode = NamespaceModeAll
	}

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

	// Validate name format (lowercase, alphanumeric, hyphens)
	if !isValidName(rc.Name) {
		return fmt.Errorf("invalid name format: %s (must be lowercase alphanumeric with hyphens)", rc.Name)
	}

	// Validate resource type format
	if !isValidResourceType(rc.ResourceType) {
		return fmt.Errorf("invalid resource_type format: %s (must be lowercase alphanumeric with underscores)", rc.ResourceType)
	}

	// Validate nicknames
	for _, nickname := range rc.Nicknames {
		if !isValidName(nickname) {
			return fmt.Errorf("invalid nickname format: %s (must be lowercase alphanumeric with hyphens)", nickname)
		}
	}

	// Validate supported clusters
	if len(rc.SupportedClusters) == 0 {
		return fmt.Errorf("at least one supported cluster type must be specified")
	}

	for _, cluster := range rc.SupportedClusters {
		if cluster != ClusterTypeKubernetes && cluster != ClusterTypeOpenShift {
			return fmt.Errorf("invalid cluster type: %s (valid: kubernetes, openshift)", cluster)
		}
	}

	// Validate namespaced vs cluster_scoped consistency
	if rc.Namespaced && rc.ClusterScoped {
		return fmt.Errorf("resource cannot be both namespaced and cluster_scoped")
	}

	// Validate rate limit
	if rc.RateLimit < 0 {
		return fmt.Errorf("rate_limit must be >= 0")
	}

	return nil
}

// SetDefaults sets default values for a resource collection
func (rc *ResourceCollection) SetDefaults() {
	// Default: enabled
	// (We don't set this here because the zero value for bool is false,
	// and we want to allow explicit false values. The loader should handle this.)

	// Set default rate limit from global settings if not specified
	if rc.RateLimit == 0 {
		rc.RateLimit = 10 // Default per-resource rate limit
	}

	// If neither namespaced nor cluster_scoped is specified, default based on common patterns
	if !rc.Namespaced && !rc.ClusterScoped {
		// Common cluster-scoped resources
		clusterScopedResources := []string{"nodes", "persistentvolumes", "clusterroles", "clusterrolebindings", "crds", "namespaces"}
		for _, csr := range clusterScopedResources {
			if strings.Contains(strings.ToLower(rc.Name), csr) {
				rc.ClusterScoped = true
				return
			}
		}
		// Default to namespaced
		rc.Namespaced = true
	}
}

// GetByName returns a collection by name or nickname
func (c *CollectionsConfig) GetByName(name string) *ResourceCollection {
	for i := range c.Collections {
		if c.Collections[i].Name == name {
			return &c.Collections[i]
		}
		for _, nickname := range c.Collections[i].Nicknames {
			if nickname == name {
				return &c.Collections[i]
			}
		}
	}
	return nil
}

// GetByResourceType returns a collection by resource type
func (c *CollectionsConfig) GetByResourceType(resourceType string) *ResourceCollection {
	for i := range c.Collections {
		if c.Collections[i].ResourceType == resourceType {
			return &c.Collections[i]
		}
	}
	return nil
}

// GetEnabledCollections returns all enabled collections
func (c *CollectionsConfig) GetEnabledCollections() []ResourceCollection {
	var enabled []ResourceCollection
	for _, collection := range c.Collections {
		if collection.Enabled {
			enabled = append(enabled, collection)
		}
	}
	return enabled
}

// GetCollectionsForCluster returns collections supported by the given cluster type
func (c *CollectionsConfig) GetCollectionsForCluster(clusterType ClusterType) []ResourceCollection {
	var supported []ResourceCollection
	for _, collection := range c.Collections {
		if collection.SupportsCluster(clusterType) {
			supported = append(supported, collection)
		}
	}
	return supported
}

// SupportsCluster checks if this collection supports the given cluster type
func (rc *ResourceCollection) SupportsCluster(clusterType ClusterType) bool {
	for _, supported := range rc.SupportedClusters {
		if supported == clusterType {
			return true
		}
	}
	return false
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
