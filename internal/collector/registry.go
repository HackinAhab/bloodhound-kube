package collector

import (
	"fmt"
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
	r.Register(NewNodesHandler())
	r.Register(NewSecretsHandler())
	r.Register(NewServicesHandler())
	r.Register(NewIngressesHandler())
	r.Register(NewGatewaysHandler())
	r.Register(NewRoutesHandler())
	r.Register(NewRbacHandler())
	r.Register(NewConfigMapsHandler())
	r.Register(NewNetworkPoliciesHandler())
	r.Register(NewCRDHandler())
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

var DefaultRegistry = NewResourceRegistry()