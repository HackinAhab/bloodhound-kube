package config

import (
	"bloodhound-kube/internal/utils"
	"fmt"
)

// ConfigVersion represents the version of the configuration schema
type ConfigVersion string

const (
	ConfigVersion1_0 ConfigVersion = "1.0"
)

// ConfigMetadata contains metadata about a configuration file
type ConfigMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version,omitempty"`
}

// Validate checks if the metadata is valid
func (m *ConfigMetadata) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	return nil
}

// ClusterType represents the type of Kubernetes cluster
type ClusterType string

const (
	ClusterTypeKubernetes ClusterType = "kubernetes"
	ClusterTypeOpenShift  ClusterType = "openshift"
)

// ParseClusterType converts a string to a ClusterType
func ParseClusterType(s string) (ClusterType, error) {
	switch s {
	case "kubernetes":
		return ClusterTypeKubernetes, nil
	case "openshift":
		return ClusterTypeOpenShift, nil
	default:
		return "", fmt.Errorf("unknown cluster type: %s (valid: kubernetes, openshift)", s)
	}
}

// ToUtilsClusterType converts config ClusterType to utils.ClusterType
func (ct ClusterType) ToUtilsClusterType() utils.ClusterType {
	switch ct {
	case ClusterTypeKubernetes:
		return utils.ClusterTypeKubernetes
	case ClusterTypeOpenShift:
		return utils.ClusterTypeOpenShift
	default:
		return utils.ClusterTypeKubernetes
	}
}

// FromUtilsClusterType converts utils.ClusterType to config ClusterType
func FromUtilsClusterType(uct utils.ClusterType) ClusterType {
	switch uct {
	case utils.ClusterTypeKubernetes:
		return ClusterTypeKubernetes
	case utils.ClusterTypeOpenShift:
		return ClusterTypeOpenShift
	default:
		return ClusterTypeKubernetes
	}
}

// NamespaceMode defines how namespaces should be filtered
type NamespaceMode string

const (
	NamespaceModeAll     NamespaceMode = "all"     // Collect from all namespaces
	NamespaceModeInclude NamespaceMode = "include" // Only collect from listed namespaces
	NamespaceModeExclude NamespaceMode = "exclude" // Collect from all except listed namespaces
)

// NamespaceFilter defines namespace filtering configuration
type NamespaceFilter struct {
	Mode NamespaceMode `yaml:"mode"`
	List []string      `yaml:"list,omitempty"`
}

// Validate checks if the namespace filter is valid
func (nf *NamespaceFilter) Validate() error {
	switch nf.Mode {
	case NamespaceModeAll, NamespaceModeInclude, NamespaceModeExclude:
		// Valid modes
	case "":
		nf.Mode = NamespaceModeAll // Default
	default:
		return fmt.Errorf("invalid namespace mode: %s (valid: all, include, exclude)", nf.Mode)
	}

	if (nf.Mode == NamespaceModeInclude || nf.Mode == NamespaceModeExclude) && len(nf.List) == 0 {
		return fmt.Errorf("namespace list cannot be empty when mode is %s", nf.Mode)
	}

	return nil
}

// ShouldCollectNamespace checks if a namespace should be collected based on the filter
func (nf *NamespaceFilter) ShouldCollectNamespace(namespace string) bool {
	switch nf.Mode {
	case NamespaceModeAll:
		return true
	case NamespaceModeInclude:
		return contains(nf.List, namespace)
	case NamespaceModeExclude:
		return !contains(nf.List, namespace)
	default:
		return true
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// PerformanceSettings defines performance-related configuration
type PerformanceSettings struct {
	ParallelCollectors int `yaml:"parallel_collectors,omitempty"`
	BatchSize          int `yaml:"batch_size,omitempty"`
	TimeoutSeconds     int `yaml:"timeout_seconds,omitempty"`
	RateLimitPerSecond int `yaml:"rate_limit_per_second,omitempty"`
}

// Validate checks if performance settings are valid
func (ps *PerformanceSettings) Validate() error {
	if ps.ParallelCollectors < 0 {
		return fmt.Errorf("parallel_collectors must be >= 0")
	}
	if ps.BatchSize < 0 {
		return fmt.Errorf("batch_size must be >= 0")
	}
	if ps.TimeoutSeconds < 0 {
		return fmt.Errorf("timeout_seconds must be >= 0")
	}
	if ps.RateLimitPerSecond < 0 {
		return fmt.Errorf("rate_limit_per_second must be >= 0")
	}
	return nil
}

// SetDefaults sets default values for performance settings
func (ps *PerformanceSettings) SetDefaults() {
	if ps.ParallelCollectors == 0 {
		ps.ParallelCollectors = 5
	}
	if ps.BatchSize == 0 {
		ps.BatchSize = 100
	}
	if ps.TimeoutSeconds == 0 {
		ps.TimeoutSeconds = 300
	}
	if ps.RateLimitPerSecond == 0 {
		ps.RateLimitPerSecond = 10
	}
}
