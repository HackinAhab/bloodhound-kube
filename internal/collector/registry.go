package collector

import (
	"bloodhound-kube/internal/utils"
	"fmt"
	"slices"
	"sort"
	"strings"
)

type ResourceRegistry struct {
	handlers map[string]ResourceHandler
}

func NewResourceRegistry() *ResourceRegistry {
	registry := &ResourceRegistry{
		handlers: make(map[string]ResourceHandler),
	}

	registry.registerDefaults()
	return registry
}

func (r *ResourceRegistry) registerDefaults() {
	// Register all handlers from metadata
	for _, meta := range AllHandlers {
		handler := NewHandlerFromMetadata(meta)
		r.Register(handler)
	}
}

func (r *ResourceRegistry) InitializeForCluster(clusterType utils.ClusterType) {
	// Clear existing handlers
	r.handlers = make(map[string]ResourceHandler)

	// Register handlers that support the current cluster type
	for _, meta := range AllHandlers {
		if r.supportsClusterType(meta.SupportedClusterTypes, clusterType) {
			handler := NewHandlerFromMetadata(meta)
			r.Register(handler)
		}
	}
}

// supportsClusterType checks if the handler supports the given cluster type
func (r *ResourceRegistry) supportsClusterType(supportedTypes []utils.ClusterType, clusterType utils.ClusterType) bool {
	return slices.Contains(supportedTypes, clusterType)
}

func (r *ResourceRegistry) Register(handler ResourceHandler) {
	r.handlers[handler.GetName()] = handler
}

func (r *ResourceRegistry) GetHandler(name string) (ResourceHandler, error) {
	handler, exists := r.handlers[name]
	if !exists {
		return nil, fmt.Errorf("unknown resource type: %s", name)
	}
	return handler, nil
}

func (r *ResourceRegistry) GetAllNames() []string {
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *ResourceRegistry) GetNamespacedTypes(names []string) []string {
	var namespaced []string
	for _, name := range names {
		if handler, exists := r.handlers[name]; exists && !handler.IsClusterScoped() {
			namespaced = append(namespaced, name)
		}
	}
	return namespaced
}

func (r *ResourceRegistry) GetClusterScopedTypes(names []string) []string {
	var clusterScoped []string
	for _, name := range names {
		if handler, exists := r.handlers[name]; exists && handler.IsClusterScoped() {
			clusterScoped = append(clusterScoped, name)
		}
	}
	return clusterScoped
}

func (r *ResourceRegistry) ValidateTypes(types []string) error {
	var invalid []string
	for _, t := range types {
		if _, exists := r.handlers[t]; !exists {
			invalid = append(invalid, t)
		}
	}

	if len(invalid) > 0 {
		available := strings.Join(r.GetAllNames(), ", ")
		return fmt.Errorf("unsupported resource types: %s (available: %s)",
			strings.Join(invalid, ", "), available)
	}

	return nil
}

// GetHandlerDescriptions returns a map of handler names to their descriptions
func (r *ResourceRegistry) GetHandlerDescriptions() map[string]string {
	descriptions := make(map[string]string)
	for name, handler := range r.handlers {
		descriptions[name] = handler.GetDescription()
	}
	return descriptions
}

// ListHandlersForClusterType returns all handlers that support the given cluster type
func (r *ResourceRegistry) ListHandlersForClusterType(clusterType utils.ClusterType) []string {
	var handlers []string
	for _, meta := range AllHandlers {
		if r.supportsClusterType(meta.SupportedClusterTypes, clusterType) {
			handlers = append(handlers, meta.Name)
		}
	}
	sort.Strings(handlers)
	return handlers
}

// GetAvailableHandlers returns metadata for all available handlers
func (r *ResourceRegistry) GetAvailableHandlers() []HandlerMetadata {
	return AllHandlers
}

var DefaultRegistry = NewResourceRegistry()
