package config

import (
	"fmt"
	"strings"
)

// ParsersConfig represents the root configuration for resource parsing
type ParsersConfig struct {
	Version    string             `yaml:"version"`
	Metadata   ConfigMetadata     `yaml:"metadata"`
	Transforms map[string]string  `yaml:"transforms,omitempty"` // Built-in transform descriptions
	Parsers    []ParserDefinition `yaml:"parsers"`
}

// ParserDefinition defines how to parse a specific resource type
type ParserDefinition struct {
	ResourceType    string               `yaml:"resource_type"`
	BloodHoundKinds []string             `yaml:"bloodhound_kinds"`
	Enabled         bool                 `yaml:"enabled"`
	Properties      []PropertyDefinition `yaml:"properties"`
}

// PropertyDefinition defines how to extract or compute a property
type PropertyDefinition struct {
	Name       string `yaml:"name"`
	Source     string `yaml:"source,omitempty"`     // Dot-notation path (e.g., "metadata.name")
	Value      any    `yaml:"value,omitempty"`      // Static value
	Expression string `yaml:"expression,omitempty"` // Computed expression (e.g., "len(data)")
	Transform  string `yaml:"transform,omitempty"`  // Transform function name
	Required   bool   `yaml:"required,omitempty"`   // Whether this property is required
	Default    any    `yaml:"default,omitempty"`    // Default value if not found
}

// Validate checks if the parsers configuration is valid
func (p *ParsersConfig) Validate() error {
	// Validate version
	if p.Version == "" {
		return fmt.Errorf("version is required")
	}
	if p.Version != string(ConfigVersion1_0) {
		return fmt.Errorf("unsupported config version: %s (supported: %s)", p.Version, ConfigVersion1_0)
	}

	// Validate metadata
	if err := p.Metadata.Validate(); err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}

	// Validate parsers
	if len(p.Parsers) == 0 {
		return fmt.Errorf("at least one parser must be defined")
	}

	// Track resource types for duplicate detection
	resourceTypes := make(map[string]bool)

	for i, parser := range p.Parsers {
		if err := parser.Validate(); err != nil {
			return fmt.Errorf("parser[%d] (%s) validation failed: %w", i, parser.ResourceType, err)
		}

		// Check for duplicate resource types
		if resourceTypes[parser.ResourceType] {
			return fmt.Errorf("duplicate parser for resource type: %s", parser.ResourceType)
		}
		resourceTypes[parser.ResourceType] = true
	}

	return nil
}

// SetDefaults sets default values for the configuration
func (p *ParsersConfig) SetDefaults() {
	// Set defaults for each parser
	for i := range p.Parsers {
		p.Parsers[i].SetDefaults()
	}
}

// Validate checks if a parser definition is valid
func (pd *ParserDefinition) Validate() error {
	// Required fields
	if pd.ResourceType == "" {
		return fmt.Errorf("resource_type is required")
	}

	// Validate resource type format
	if !isValidResourceType(pd.ResourceType) {
		return fmt.Errorf("invalid resource_type format: %s (must be lowercase alphanumeric with underscores)", pd.ResourceType)
	}

	// Validate BloodHound kinds
	if len(pd.BloodHoundKinds) == 0 {
		return fmt.Errorf("at least one bloodhound_kind must be specified")
	}

	for _, kind := range pd.BloodHoundKinds {
		if !isValidBloodHoundKind(kind) {
			return fmt.Errorf("invalid bloodhound_kind format: %s (must be PascalCase)", kind)
		}
	}

	// Validate properties
	if len(pd.Properties) == 0 {
		return fmt.Errorf("at least one property must be defined")
	}

	// Track property names for duplicate detection
	propertyNames := make(map[string]bool)

	for i, prop := range pd.Properties {
		if err := prop.Validate(); err != nil {
			return fmt.Errorf("property[%d] (%s) validation failed: %w", i, prop.Name, err)
		}

		// Check for duplicate property names
		if propertyNames[prop.Name] {
			return fmt.Errorf("duplicate property name: %s", prop.Name)
		}
		propertyNames[prop.Name] = true
	}

	return nil
}

// SetDefaults sets default values for a parser definition
func (pd *ParserDefinition) SetDefaults() {
	// Default: enabled
	// (We don't set this here because the zero value for bool is false,
	// and we want to allow explicit false values. The loader should handle this.)

	// Set defaults for each property
	for i := range pd.Properties {
		pd.Properties[i].SetDefaults()
	}
}

// Validate checks if a property definition is valid
func (prop *PropertyDefinition) Validate() error {
	// Required fields
	if prop.Name == "" {
		return fmt.Errorf("property name is required")
	}

	// Validate property name format (lowercase with underscores)
	if !isValidPropertyName(prop.Name) {
		return fmt.Errorf("invalid property name format: %s (must be lowercase alphanumeric with underscores)", prop.Name)
	}

	// Count how many value sources are specified
	sourceCount := 0
	if prop.Source != "" {
		sourceCount++
	}
	if prop.Value != nil {
		sourceCount++
	}
	if prop.Expression != "" {
		sourceCount++
	}

	// Exactly one source must be specified
	if sourceCount == 0 {
		return fmt.Errorf("property %s must have one of: source, value, or expression", prop.Name)
	}
	if sourceCount > 1 {
		return fmt.Errorf("property %s can only have one of: source, value, or expression", prop.Name)
	}

	// Validate source path format (if specified)
	if prop.Source != "" {
		if err := validateSourcePath(prop.Source); err != nil {
			return fmt.Errorf("invalid source path for property %s: %w", prop.Name, err)
		}
	}

	// Validate expression format (if specified)
	if prop.Expression != "" {
		if err := validateExpression(prop.Expression); err != nil {
			return fmt.Errorf("invalid expression for property %s: %w", prop.Name, err)
		}
	}

	// Transform can only be used with source
	if prop.Transform != "" && prop.Source == "" {
		return fmt.Errorf("property %s: transform can only be used with source", prop.Name)
	}

	return nil
}

// SetDefaults sets default values for a property definition
func (prop *PropertyDefinition) SetDefaults() {
	// No defaults to set currently
}

// GetByResourceType returns a parser by resource type
func (p *ParsersConfig) GetByResourceType(resourceType string) *ParserDefinition {
	for i := range p.Parsers {
		if p.Parsers[i].ResourceType == resourceType {
			return &p.Parsers[i]
		}
	}
	return nil
}

// GetEnabledParsers returns all enabled parsers
func (p *ParsersConfig) GetEnabledParsers() []ParserDefinition {
	var enabled []ParserDefinition
	for _, parser := range p.Parsers {
		if parser.Enabled {
			enabled = append(enabled, parser)
		}
	}
	return enabled
}

// GetRequiredProperties returns all required properties for this parser
func (pd *ParserDefinition) GetRequiredProperties() []PropertyDefinition {
	var required []PropertyDefinition
	for _, prop := range pd.Properties {
		if prop.Required {
			required = append(required, prop)
		}
	}
	return required
}

// HasProperty checks if this parser has a property with the given name
func (pd *ParserDefinition) HasProperty(name string) bool {
	for _, prop := range pd.Properties {
		if prop.Name == name {
			return true
		}
	}
	return false
}

// isValidBloodHoundKind checks if a BloodHound kind is valid (PascalCase with optional underscores)
func isValidBloodHoundKind(kind string) bool {
	if kind == "" {
		return false
	}
	// Must start with uppercase letter
	if kind[0] < 'A' || kind[0] > 'Z' {
		return false
	}
	// Can contain letters, numbers, and underscores
	for _, c := range kind {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// isValidPropertyName checks if a property name is valid (lowercase, alphanumeric, underscores)
func isValidPropertyName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// validateSourcePath validates a dot-notation source path
func validateSourcePath(path string) error {
	if path == "" {
		return fmt.Errorf("source path cannot be empty")
	}

	// Split by dots
	parts := strings.Split(path, ".")
	for i, part := range parts {
		if part == "" {
			return fmt.Errorf("empty path component at position %d", i)
		}

		// Check for array indexing (e.g., "containers[0]" or "containers[*]")
		if strings.Contains(part, "[") {
			if !strings.HasSuffix(part, "]") {
				return fmt.Errorf("invalid array syntax in: %s", part)
			}
			// Valid array syntax: name[index] or name[*]
			bracketIndex := strings.Index(part, "[")
			fieldName := part[:bracketIndex]
			indexPart := part[bracketIndex+1 : len(part)-1]

			// Validate field name
			if !isValidFieldName(fieldName) {
				return fmt.Errorf("invalid field name: %s", fieldName)
			}

			// Validate index (must be number or *)
			if indexPart != "*" {
				for _, c := range indexPart {
					if c < '0' || c > '9' {
						return fmt.Errorf("invalid array index: %s (must be number or *)", indexPart)
					}
				}
			}
		} else {
			// Regular field name
			if !isValidFieldName(part) {
				return fmt.Errorf("invalid field name: %s", part)
			}
		}
	}

	return nil
}

// validateExpression validates a simple expression
func validateExpression(expr string) error {
	if expr == "" {
		return fmt.Errorf("expression cannot be empty")
	}

	// Basic validation: check for common expression patterns
	// This is a simple check - the actual expression evaluator will do more thorough validation

	// Check for balanced parentheses
	depth := 0
	for _, c := range expr {
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
			if depth < 0 {
				return fmt.Errorf("unbalanced parentheses in expression")
			}
		}
	}
	if depth != 0 {
		return fmt.Errorf("unbalanced parentheses in expression")
	}

	// Check for balanced brackets
	depth = 0
	for _, c := range expr {
		if c == '[' {
			depth++
		} else if c == ']' {
			depth--
			if depth < 0 {
				return fmt.Errorf("unbalanced brackets in expression")
			}
		}
	}
	if depth != 0 {
		return fmt.Errorf("unbalanced brackets in expression")
	}

	return nil
}

// isValidFieldName checks if a field name is valid (alphanumeric, underscores, first char not number)
func isValidFieldName(name string) bool {
	if name == "" {
		return false
	}
	// First character cannot be a number
	if name[0] >= '0' && name[0] <= '9' {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}
