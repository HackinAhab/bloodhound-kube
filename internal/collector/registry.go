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

func (r *ResourceRegistry) ValidateTypes(types []string) error {
	var invalid []string
	for _, t := range types {
		if _, err := r.GetHandler(t); err != nil {
			invalid = append(invalid, t)
		}
	}

	if len(invalid) > 0 {
		return r.unsupportedTypesError(invalid)
	}

	return nil
}

func (r *ResourceRegistry) NormalizeTypes(types []string) ([]string, error) {
	if len(types) == 0 {
		return types, nil
	}

	lookup, ambiguous := r.buildAliasLookup()
	resolved := make([]string, 0, len(types))
	var invalid []string

	for _, t := range types {
		canonical, ok := resolveTypeWithLookup(t, lookup, ambiguous)
		if ok {
			resolved = append(resolved, canonical)
		} else {
			invalid = append(invalid, t)
		}
	}

	if len(invalid) > 0 {
		return nil, r.unsupportedTypesError(invalid)
	}

	return resolved, nil
}

func (r *ResourceRegistry) ResolveType(input string) (string, bool) {
	if input == "" {
		return "", false
	}

	lookup, ambiguous := r.buildAliasLookup()
	return resolveTypeWithLookup(input, lookup, ambiguous)
}

// getAvailableTypesWithNicknames returns a formatted string of available types including nicknames
func (r *ResourceRegistry) getAvailableTypesWithNicknames() string {
	resources := r.getAvailableTypesDisplay()
	return strings.Join(resources, ", ")
}

func (r *ResourceRegistry) buildAliasLookup() (map[string]string, map[string]struct{}) {
	lookup := make(map[string]string)
	ambiguous := make(map[string]struct{})

	addAlias := func(alias, canonical string) {
		if alias == "" || canonical == "" {
			return
		}
		key := normalizeTypeKey(alias)
		if key == "" {
			return
		}
		if existing, ok := lookup[key]; ok {
			if existing != canonical {
				delete(lookup, key)
				ambiguous[key] = struct{}{}
			}
			return
		}
		if _, ok := ambiguous[key]; ok {
			return
		}
		lookup[key] = canonical
	}

	if r.factory != nil && r.factory.config != nil {
		for _, collection := range r.factory.config.Collections {
			if !collection.Enabled {
				continue
			}
			canonical := collection.Name
			addAlias(collection.Name, canonical)
			addAlias(collection.Kind, canonical)
			for _, shortName := range collection.ShortNames {
				addAlias(shortName, canonical)
			}
			addAlias(collection.APIPath, canonical)
		}
		return lookup, ambiguous
	}

	seenHandlers := make(map[ResourceHandler]struct{})
	for _, handler := range r.handlers {
		if _, exists := seenHandlers[handler]; exists {
			continue
		}
		seenHandlers[handler] = struct{}{}
		addAlias(handler.GetName(), handler.GetName())
	}

	return lookup, ambiguous
}

func (r *ResourceRegistry) unsupportedTypesError(invalid []string) error {
	available := r.getAvailableTypesWithNicknames()
	return fmt.Errorf("unsupported resource types: %s (available: %s)",
		strings.Join(invalid, ", "), available)
}

func resolveTypeWithLookup(input string, lookup map[string]string, ambiguous map[string]struct{}) (string, bool) {
	normalizedInput := normalizeTypeKey(input)
	if normalizedInput == "" {
		return "", false
	}
	if _, exists := ambiguous[normalizedInput]; exists {
		return "", false
	}
	canonical, ok := lookup[normalizedInput]
	return canonical, ok
}

func normalizeTypeKey(value string) string {
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case c >= 'A' && c <= 'Z':
			b.WriteByte(c + ('a' - 'A'))
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'):
			b.WriteByte(c)
		}
	}
	return b.String()
}

func (r *ResourceRegistry) getAvailableTypesDisplay() []string {
	if r.factory != nil && r.factory.config != nil {
		resources := make([]string, 0, len(r.factory.config.Collections))
		seen := make(map[string]struct{})
		for _, collection := range r.factory.config.Collections {
			if !collection.Enabled {
				continue
			}
			entry := collection.Kind
			if entry == "" {
				entry = collection.Name
			}
			if _, exists := seen[entry]; exists {
				continue
			}
			seen[entry] = struct{}{}
			resources = append(resources, entry)
		}
		sort.Strings(resources)
		return resources
	}

	resources := make([]string, 0, len(r.handlers))
	for name, handler := range r.handlers {
		desc := handler.GetDescription()
		if desc != "" {
			resources = append(resources, fmt.Sprintf("%s (%s)", name, desc))
		} else {
			resources = append(resources, name)
		}
	}
	if len(resources) > 0 {
		sort.Strings(resources)
	}
	return resources
}

var DefaultRegistry = NewResourceRegistry()
