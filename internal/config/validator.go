package config

import (
	"fmt"
	"strings"
)

// ValidationError represents a configuration validation error with context
type ValidationError struct {
	Field   string // Field name or path
	Message string // Error message
	Line    int    // Line number in YAML (if available)
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	if ve.Line > 0 {
		return fmt.Sprintf("line %d, field %s: %s", ve.Line, ve.Field, ve.Message)
	}
	return fmt.Sprintf("field %s: %s", ve.Field, ve.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface for multiple errors
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "no validation errors"
	}
	if len(ve) == 1 {
		return ve[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d validation errors:\n", len(ve)))
	for i, err := range ve {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// HasErrors returns true if there are any validation errors
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// Validator provides validation utilities with detailed error reporting
type Validator struct {
	errors ValidationErrors
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

// AddError adds a validation error
func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// AddErrorf adds a formatted validation error
func (v *Validator) AddErrorf(field, format string, args ...any) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: fmt.Sprintf(format, args...),
	})
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors returns all validation errors
func (v *Validator) Errors() ValidationErrors {
	return v.errors
}

// Error returns the error message (implements error interface)
func (v *Validator) Error() error {
	if !v.HasErrors() {
		return nil
	}
	return v.errors
}

// ValidateCollectionsConfig performs detailed validation on a collections config
func ValidateCollectionsConfig(config *CollectionsConfig) error {
	validator := NewValidator()

	// Version validation
	if config.Version == "" {
		validator.AddError("version", "version is required")
	} else if config.Version != string(ConfigVersion1_0) {
		validator.AddErrorf("version", "unsupported version %s (supported: %s)", config.Version, ConfigVersion1_0)
	}

	// Metadata validation
	if config.Metadata.Name == "" {
		validator.AddError("metadata.name", "name is required")
	}

	// Settings validation
	if config.Settings.ParallelCollectors < 0 {
		validator.AddError("settings.parallel_collectors", "must be >= 0")
	}
	if config.Settings.BatchSize < 0 {
		validator.AddError("settings.batch_size", "must be >= 0")
	}
	if config.Settings.TimeoutSeconds < 0 {
		validator.AddError("settings.timeout_seconds", "must be >= 0")
	}

	// Namespace validation
	switch config.Namespaces.Mode {
	case NamespaceModeAll, NamespaceModeInclude, NamespaceModeExclude:
		// Valid
	case "":
		// Will be set to default
	default:
		validator.AddErrorf("namespaces.mode", "invalid mode %s (valid: all, include, exclude)", config.Namespaces.Mode)
	}

	// Collections validation
	if len(config.Collections) == 0 {
		validator.AddError("collections", "at least one collection must be defined")
	}

	names := make(map[string]bool)
	resourceTypes := make(map[string]bool)

	for i, collection := range config.Collections {
		prefix := fmt.Sprintf("collections[%d]", i)

		// Name validation
		if collection.Name == "" {
			validator.AddErrorf(prefix+".name", "name is required")
		} else {
			if names[collection.Name] {
				validator.AddErrorf(prefix+".name", "duplicate name: %s", collection.Name)
			}
			names[collection.Name] = true

			if !isValidName(collection.Name) {
				validator.AddErrorf(prefix+".name", "invalid format: %s (must be lowercase alphanumeric with hyphens)", collection.Name)
			}
		}

		// Resource type validation
		if collection.ResourceType == "" {
			validator.AddErrorf(prefix+".resource_type", "resource_type is required")
		} else {
			if resourceTypes[collection.ResourceType] {
				validator.AddErrorf(prefix+".resource_type", "duplicate resource type: %s", collection.ResourceType)
			}
			resourceTypes[collection.ResourceType] = true

			if !isValidResourceType(collection.ResourceType) {
				validator.AddErrorf(prefix+".resource_type", "invalid format: %s", collection.ResourceType)
			}
		}

		// API validation
		if collection.APIVersion == "" {
			validator.AddErrorf(prefix+".api_version", "api_version is required")
		}
		if collection.Plural == "" {
			validator.AddErrorf(prefix+".plural", "plural is required")
		}

		// Cluster support validation
		if len(collection.SupportedClusters) == 0 {
			validator.AddErrorf(prefix+".supported_clusters", "at least one cluster type required")
		}

		// Scope validation
		if collection.Namespaced && collection.ClusterScoped {
			validator.AddErrorf(prefix, "cannot be both namespaced and cluster_scoped")
		}
	}

	return validator.Error()
}

// ValidateParsersConfig performs detailed validation on a parsers config
func ValidateParsersConfig(config *ParsersConfig) error {
	validator := NewValidator()

	// Version validation
	if config.Version == "" {
		validator.AddError("version", "version is required")
	} else if config.Version != string(ConfigVersion1_0) {
		validator.AddErrorf("version", "unsupported version %s (supported: %s)", config.Version, ConfigVersion1_0)
	}

	// Metadata validation
	if config.Metadata.Name == "" {
		validator.AddError("metadata.name", "name is required")
	}

	// Parsers validation
	if len(config.Parsers) == 0 {
		validator.AddError("parsers", "at least one parser must be defined")
	}

	resourceTypes := make(map[string]bool)

	for i, parser := range config.Parsers {
		prefix := fmt.Sprintf("parsers[%d]", i)

		// Resource type validation
		if parser.ResourceType == "" {
			validator.AddErrorf(prefix+".resource_type", "resource_type is required")
		} else {
			if resourceTypes[parser.ResourceType] {
				validator.AddErrorf(prefix+".resource_type", "duplicate parser for: %s", parser.ResourceType)
			}
			resourceTypes[parser.ResourceType] = true

			if !isValidResourceType(parser.ResourceType) {
				validator.AddErrorf(prefix+".resource_type", "invalid format: %s", parser.ResourceType)
			}
		}

		// BloodHound kinds validation
		if len(parser.BloodHoundKinds) == 0 {
			validator.AddErrorf(prefix+".bloodhound_kinds", "at least one kind required")
		}

		for j, kind := range parser.BloodHoundKinds {
			if !isValidBloodHoundKind(kind) {
				validator.AddErrorf(prefix+fmt.Sprintf(".bloodhound_kinds[%d]", j), "invalid format: %s (must be PascalCase)", kind)
			}
		}

		// Properties validation
		if len(parser.Properties) == 0 {
			validator.AddErrorf(prefix+".properties", "at least one property required")
		}

		propertyNames := make(map[string]bool)

		for j, prop := range parser.Properties {
			propPrefix := fmt.Sprintf("%s.properties[%d]", prefix, j)

			// Name validation
			if prop.Name == "" {
				validator.AddErrorf(propPrefix+".name", "name is required")
			} else {
				if propertyNames[prop.Name] {
					validator.AddErrorf(propPrefix+".name", "duplicate property: %s", prop.Name)
				}
				propertyNames[prop.Name] = true

				if !isValidPropertyName(prop.Name) {
					validator.AddErrorf(propPrefix+".name", "invalid format: %s", prop.Name)
				}
			}

			// Source validation
			sourceCount := 0
			if prop.Source != "" {
				sourceCount++
				if err := validateSourcePath(prop.Source); err != nil {
					validator.AddErrorf(propPrefix+".source", "invalid path: %v", err)
				}
			}
			if prop.Value != nil {
				sourceCount++
			}
			if prop.Expression != "" {
				sourceCount++
				if err := validateExpression(prop.Expression); err != nil {
					validator.AddErrorf(propPrefix+".expression", "invalid expression: %v", err)
				}
			}

			if sourceCount == 0 {
				validator.AddErrorf(propPrefix, "must specify one of: source, value, or expression")
			} else if sourceCount > 1 {
				validator.AddErrorf(propPrefix, "can only specify one of: source, value, or expression")
			}

			// Transform validation
			if prop.Transform != "" && prop.Source == "" {
				validator.AddErrorf(propPrefix+".transform", "can only be used with source")
			}
		}
	}

	return validator.Error()
}

// SuggestFix provides helpful suggestions for common validation errors
func SuggestFix(err error) string {
	errMsg := err.Error()

	// Common error patterns and their fixes
	suggestions := map[string]string{
		"version is required":                          "Add: version: \"1.0\"",
		"metadata.name is required":                    "Add: meta\n  name: \"your-config-name\"",
		"invalid format":                               "Use lowercase letters, numbers, and underscores only",
		"must be PascalCase":                           "Start with uppercase letter, no spaces or special characters",
		"at least one collection must be defined":      "Add at least one collection in the collections array",
		"at least one parser must be defined":          "Add at least one parser in the parsers array",
		"duplicate":                                    "Remove or rename the duplicate entry",
		"cannot be both namespaced and cluster_scoped": "Set only one to true",
	}

	for pattern, suggestion := range suggestions {
		if strings.Contains(errMsg, pattern) {
			return suggestion
		}
	}

	return "Check the configuration documentation for the correct format"
}
