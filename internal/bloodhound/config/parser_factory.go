// This file implements the parser factory for creating parsers from YAML configuration.
// It integrates with the existing parser registry and uses the property engine
// and expression evaluator to create dynamic, YAML-driven parsers.

package config

import (
	"fmt"
	"os"
	"strings"

	"bloodhound-kube/internal/bloodhound"
	"bloodhound-kube/internal/config"
)

// ConfigurableParser is a parser that can be configured via YAML
type ConfigurableParser struct {
	config        config.ParserDefinition
	engine        *PropertyEngine
	resourceTypes []string
	primaryKind   string
}

// NewConfigurableParser creates a new parser from YAML configuration
func NewConfigurableParser(cfg config.ParserDefinition) (*ConfigurableParser, error) {
	// Validate configuration
	if cfg.ResourceType == "" {
		return nil, fmt.Errorf("resource_type is required")
	}
	if len(cfg.Properties) == 0 {
		return nil, fmt.Errorf("at least one property must be defined")
	}

	// Create property engine
	engine := NewPropertyEngine()

	// Determine kinds (use BloodHoundKinds, or capitalize resource_type as fallback)
	kinds := cfg.BloodHoundKinds
	if len(kinds) == 0 {
		// Default: use capitalized resource_type
		kinds = []string{capitalizeFirst(cfg.ResourceType)}
	}

	return &ConfigurableParser{
		config:        cfg,
		engine:        engine,
		resourceTypes: []string{cfg.ResourceType},
		primaryKind:   kinds[0], // First kind is primary
	}, nil
}

// GetResourceType implements Parser interface
func (p *ConfigurableParser) GetResourceType() string {
	return p.config.ResourceType
}

// Parse implements Parser interface
func (p *ConfigurableParser) Parse(resource bloodhound.ResourceData) (*bloodhound.ParsedResult, error) {
	// Convert resource to map for property extraction
	var resourceMap map[string]any
	switch v := resource.Resource.(type) {
	case map[string]any:
		resourceMap = v
	default:
		return nil, fmt.Errorf("expected resource to be map[string]any, got %T", resource.Resource)
	}

	// Extract properties using the property engine
	properties, err := p.engine.ExtractProperties(resourceMap, p.config)
	if err != nil {
		return nil, fmt.Errorf("failed to extract properties: %w", err)
	}

	// Add standard properties
	properties["resource_type"] = p.config.ResourceType
	properties["namespace"] = resource.Namespace

	// Determine node name - default to "name" property
	nameField := "name"

	nodeName, ok := properties[nameField].(string)
	if !ok {
		return nil, fmt.Errorf("node name field '%s' not found or not a string", nameField)
	}

	// Create BloodHound node
	node := bloodhound.CreateNodeFromResource(
		p.config.BloodHoundKinds,
		p.config.ResourceType,
		resource.Namespace,
		nodeName,
		properties,
	)

	return &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{node},
		Edges: []bloodhound.BloodHoundEdge{},
	}, nil
}

// GetSupportedKinds implements Parser interface
func (p *ConfigurableParser) GetSupportedKinds() []string {
	return p.config.BloodHoundKinds
}

// GetConfig implements Parser interface
func (p *ConfigurableParser) GetConfig() bloodhound.ResourceConfig {
	return bloodhound.ResourceConfig{
		ResourceType: p.config.ResourceType,
		PrimaryKind:  p.primaryKind,
	}
}

// ParserFactory creates parsers from YAML configuration
type ParserFactory struct {
	parsers map[string]*ConfigurableParser
}

// NewParserFactory creates a new parser factory
func NewParserFactory() *ParserFactory {
	return &ParserFactory{
		parsers: make(map[string]*ConfigurableParser),
	}
}

// LoadParsers loads parsers from configuration
func (f *ParserFactory) LoadParsers(configs []config.ParserDefinition) error {
	for _, cfg := range configs {
		// Skip disabled parsers
		if !cfg.Enabled {
			continue
		}

		parser, err := NewConfigurableParser(cfg)
		if err != nil {
			return fmt.Errorf("failed to create parser for %s: %w", cfg.ResourceType, err)
		}

		f.parsers[cfg.ResourceType] = parser
	}

	return nil
}

// GetParser retrieves a parser by resource type
func (f *ParserFactory) GetParser(resourceType string) (*ConfigurableParser, bool) {
	parser, exists := f.parsers[resourceType]
	return parser, exists
}

// RegisterWithRegistry registers all loaded parsers with the BloodHound registry
func (f *ParserFactory) RegisterWithRegistry(registry *bloodhound.ParseRegistry) error {
	for resourceType, parser := range f.parsers {
		// Check if a Go parser already exists for this type
		if _, exists := registry.GetParser(resourceType); exists {
			// YAML parsers take precedence - override
			fmt.Printf("Overriding Go parser for %s with YAML parser\n", resourceType)
		}

		// Register the YAML parser
		registry.Register(parser)
	}

	return nil
}

// GetRegisteredTypes returns all registered parser types
func (f *ParserFactory) GetRegisteredTypes() []string {
	types := make([]string, 0, len(f.parsers))
	for resourceType := range f.parsers {
		types = append(types, resourceType)
	}
	return types
}

// Helper function to capitalize first letter
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}

	// Handle snake_case: convert to PascalCase
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		for i, part := range parts {
			if len(part) > 0 {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
		return strings.Join(parts, "")
	}

	// Simple capitalization
	return strings.ToUpper(s[:1]) + s[1:]
}

// LoadParsersFromConfig is a convenience function that loads parsers from a config file
// and registers them with the default registry
func LoadParsersFromConfig(configPath string) error {
	// Load configuration directly from file path (not using loader to avoid path issues)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	cfg, err := config.LoadParsersFromBytes(data)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create parser factory
	factory := NewParserFactory()

	// Load parsers from config
	if err := factory.LoadParsers(cfg.Parsers); err != nil {
		return fmt.Errorf("failed to load parsers: %w", err)
	}

	// Register with default registry
	if err := factory.RegisterWithRegistry(bloodhound.DefaultRegistry); err != nil {
		return fmt.Errorf("failed to register parsers: %w", err)
	}

	fmt.Printf("Loaded %d parsers from YAML configuration\n", len(factory.GetRegisteredTypes()))
	return nil
}

// LoadParsersFromConfigWithRegistry loads parsers and registers them with a specific registry
func LoadParsersFromConfigWithRegistry(configPath string, registry *bloodhound.ParseRegistry) error {
	// Load configuration directly from file path (not using loader to avoid path issues)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	cfg, err := config.LoadParsersFromBytes(data)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create parser factory
	factory := NewParserFactory()

	// Load parsers from config
	if err := factory.LoadParsers(cfg.Parsers); err != nil {
		return fmt.Errorf("failed to load parsers: %w", err)
	}

	// Register with provided registry
	if err := factory.RegisterWithRegistry(registry); err != nil {
		return fmt.Errorf("failed to register parsers: %w", err)
	}

	fmt.Printf("Loaded %d parsers from YAML configuration\n", len(factory.GetRegisteredTypes()))
	return nil
}
