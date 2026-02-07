package collector

import (
	"bloodhound-kube/internal/utils"
	"fmt"
	"sort"
	"strings"

	"k8s.io/client-go/dynamic"
)

type ResourceRegistry struct {
	handlers map[string]ResourceHandler
	factory  *CollectorFactory
}

func NewResourceRegistry() *ResourceRegistry {
	registry := &ResourceRegistry{
		handlers: make(map[string]ResourceHandler),
	}

	// Note: Registry must be initialized with InitializeFromConfig().
	return registry
}

// InitializeFromConfig initializes the registry from collection configuration.
// This takes precedence over Go handlers.
func (r *ResourceRegistry) InitializeFromConfig(clients *utils.Clients, logger *utils.Logger, collectionsConfig *CollectionsConfig, dynamicClient dynamic.Interface) error {
	factory, err := NewCollectorFactory(clients, logger, collectionsConfig, dynamicClient)
	if err != nil {
		return fmt.Errorf("failed to create collector factory: %w", err)
	}

	handlers, err := factory.CreateAllCollectors()
	if err != nil {
		return fmt.Errorf("failed to create collectors: %w", err)
	}

	// Replace existing handlers with config-defined ones
	r.handlers = handlers
	r.factory = factory

	return nil
}

func (r *ResourceRegistry) Register(handler ResourceHandler) {
	r.handlers[handler.GetName()] = handler
}

func (r *ResourceRegistry) GetHandler(name string) (ResourceHandler, error) {
	// First try exact name match
	if handler, exists := r.handlers[name]; exists {
		return handler, nil
	}

	// For config-based handlers, nicknames are handled by the factory
	// Return error with available types
	available := r.GetAllNames()
	return nil, fmt.Errorf("unknown resource type: %s (available: %s)", name, strings.Join(available, ", "))
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
		if _, err := r.GetHandler(t); err != nil {
			invalid = append(invalid, t)
		}
	}

	if len(invalid) > 0 {
		available := r.getAvailableTypesWithNicknames()
		return fmt.Errorf("unsupported resource types: %s (available: %s)",
			strings.Join(invalid, ", "), available)
	}

	return nil
}

// getAvailableTypesWithNicknames returns a formatted string of available types including nicknames
func (r *ResourceRegistry) getAvailableTypesWithNicknames() string {
	var resources []string

	// Get available resources from registered handlers
	for name, handler := range r.handlers {
		desc := handler.GetDescription()
		if desc != "" {
			resources = append(resources, fmt.Sprintf("%s (%s)", name, desc))
		} else {
			resources = append(resources, name)
		}
	}

	sort.Strings(resources)
	return strings.Join(resources, ", ")
}

// GetHandlerDescriptions returns a map of handler names to their descriptions
func (r *ResourceRegistry) GetHandlerDescriptions() map[string]string {
	descriptions := make(map[string]string)
	for name, handler := range r.handlers {
		descriptions[name] = handler.GetDescription()
	}
	return descriptions
}

var DefaultRegistry = NewResourceRegistry()
