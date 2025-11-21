package config

import (
	"bloodhound-kube/internal/config"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// PropertyEngine extracts properties from resources based on parser configuration
type PropertyEngine struct {
	transforms map[string]TransformFunc
	evaluator  *ExpressionEvaluator
}

// TransformFunc is a function that transforms a value
type TransformFunc func(value any) (any, error)

// NewPropertyEngine creates a new property extraction engine
func NewPropertyEngine() *PropertyEngine {
	engine := &PropertyEngine{
		transforms: make(map[string]TransformFunc),
		evaluator:  NewExpressionEvaluator(),
	}

	// Register built-in transforms
	engine.registerBuiltinTransforms()

	return engine
}

// ExtractProperties extracts properties from a resource based on parser definition
func (pe *PropertyEngine) ExtractProperties(resource map[string]any, parser config.ParserDefinition) (map[string]any, error) {
	properties := make(map[string]any)

	for _, propDef := range parser.Properties {
		value, err := pe.extractProperty(resource, propDef)
		if err != nil {
			if propDef.Required {
				return nil, fmt.Errorf("failed to extract required property %s: %w", propDef.Name, err)
			}
			// Use default if provided
			if propDef.Default != nil {
				properties[propDef.Name] = propDef.Default
				continue
			}
			// Skip optional properties that fail
			continue
		}

		// If the value is a map and the transform was "flatten", merge the keys into properties
		if propDef.Transform == "flatten" {
			if flatMap, ok := value.(map[string]any); ok {
				// Merge flattened keys with prefix
				for k, v := range flatMap {
					newKey := propDef.Name + "_" + k
					properties[newKey] = v
				}
				continue
			}
		}

		properties[propDef.Name] = value
	}

	return properties, nil
}

// extractProperty extracts a single property based on its definition
func (pe *PropertyEngine) extractProperty(resource map[string]any, propDef config.PropertyDefinition) (any, error) {
	// Handle static value
	if propDef.Value != nil {
		return propDef.Value, nil
	}

	// Handle source path
	if propDef.Source != "" {
		value, err := pe.extractFromSource(resource, propDef.Source)
		if err != nil {
			return nil, err
		}

		// Apply transform if specified
		if propDef.Transform != "" {
			return pe.applyTransform(value, propDef.Transform)
		}

		return value, nil
	}

	// Handle expression
	if propDef.Expression != "" {
		return pe.evaluateExpression(resource, propDef.Expression)
	}

	return nil, fmt.Errorf("property %s has no value source", propDef.Name)
}

// extractFromSource extracts a value from a resource using dot notation path
func (pe *PropertyEngine) extractFromSource(resource map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	current := any(resource)

	for i, part := range parts {
		// Handle array indexing
		if strings.Contains(part, "[") {
			fieldName, index, isWildcard, err := parseArrayAccess(part)
			if err != nil {
				return nil, fmt.Errorf("invalid array syntax in path %s: %w", path, err)
			}

			// Get the field first
			current, err = pe.getField(current, fieldName)
			if err != nil {
				return nil, fmt.Errorf("failed to access field %s: %w", fieldName, err)
			}

			// Handle array access
			if isWildcard {
				// Wildcard access: return all items at this path
				slice, ok := current.([]any)
				if !ok {
					return nil, fmt.Errorf("field %s is not an array", fieldName)
				}

				// If this is the last part, return the slice
				if i == len(parts)-1 {
					return slice, nil
				}

				// Otherwise, continue extracting from each item
				var results []any
				remainingPath := strings.Join(parts[i+1:], ".")
				for _, item := range slice {
					if itemMap, ok := item.(map[string]any); ok {
						val, err := pe.extractFromSource(itemMap, remainingPath)
						if err == nil {
							results = append(results, val)
						}
					}
				}
				return results, nil
			} else {
				// Specific index
				slice, ok := current.([]any)
				if !ok {
					return nil, fmt.Errorf("field %s is not an array", fieldName)
				}

				if index < 0 || index >= len(slice) {
					return nil, fmt.Errorf("index %d out of bounds for array %s", index, fieldName)
				}

				current = slice[index]
			}
		} else {
			// Regular field access
			var err error
			current, err = pe.getField(current, part)
			if err != nil {
				return nil, fmt.Errorf("failed to access field %s: %w", part, err)
			}
		}
	}

	return current, nil
}

// getField gets a field from a map or struct
func (pe *PropertyEngine) getField(obj any, field string) (any, error) {
	// Handle map
	if m, ok := obj.(map[string]any); ok {
		if val, exists := m[field]; exists {
			return val, nil
		}
		return nil, fmt.Errorf("field %s not found", field)
	}

	// Handle struct via reflection
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("cannot access field %s on non-map/struct type", field)
	}

	// Try to find field (case-insensitive)
	for i := 0; i < v.NumField(); i++ {
		fieldName := v.Type().Field(i).Name
		if strings.EqualFold(fieldName, field) {
			return v.Field(i).Interface(), nil
		}
	}

	return nil, fmt.Errorf("field %s not found in struct", field)
}

// parseArrayAccess parses array access syntax like "containers[0]" or "volumes[*]"
func parseArrayAccess(part string) (fieldName string, index int, isWildcard bool, err error) {
	openBracket := strings.Index(part, "[")
	closeBracket := strings.Index(part, "]")

	if openBracket == -1 || closeBracket == -1 {
		return "", 0, false, fmt.Errorf("invalid array syntax: %s", part)
	}

	fieldName = part[:openBracket]
	indexStr := part[openBracket+1 : closeBracket]

	if indexStr == "*" {
		return fieldName, 0, true, nil
	}

	index, err = strconv.Atoi(indexStr)
	if err != nil {
		return "", 0, false, fmt.Errorf("invalid array index: %s", indexStr)
	}

	return fieldName, index, false, nil
}

// applyTransform applies a transform function to a value
func (pe *PropertyEngine) applyTransform(value any, transformName string) (any, error) {
	transform, exists := pe.transforms[transformName]
	if !exists {
		return nil, fmt.Errorf("unknown transform: %s", transformName)
	}

	return transform(value)
}

// RegisterTransform registers a custom transform function
func (pe *PropertyEngine) RegisterTransform(name string, fn TransformFunc) {
	pe.transforms[name] = fn
}

// registerBuiltinTransforms registers the built-in transform functions
func (pe *PropertyEngine) registerBuiltinTransforms() {
	// map_keys: Extract keys from a map
	pe.RegisterTransform("map_keys", func(value any) (any, error) {
		m, ok := value.(map[string]any)
		if !ok {
			return []string{}, nil
		}

		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		return keys, nil
	})

	// timestamp: Parse timestamp to string (placeholder)
	pe.RegisterTransform("timestamp", func(value any) (any, error) {
		if str, ok := value.(string); ok {
			return str, nil
		}
		return value, nil
	})

	// json_string: Convert to JSON string (placeholder)
	pe.RegisterTransform("json_string", func(value any) (any, error) {
		// For now, just return as-is
		// In real implementation, would use json.Marshal
		return value, nil
	})

	// port_list: Extract port information (placeholder)
	pe.RegisterTransform("port_list", func(value any) (any, error) {
		// For now, just return as-is
		return value, nil
	})

	// flatten: Flatten nested structures into dot-notation keys
	pe.RegisterTransform("flatten", func(value any) (any, error) {
		m, ok := value.(map[string]any)
		if !ok {
			// If not a map, return as-is
			return value, nil
		}

		flattened := make(map[string]any)
		pe.flattenMap(m, "", flattened)
		return flattened, nil
	})
}

// flattenMap recursively flattens a nested map into underscore-separated keys
func (pe *PropertyEngine) flattenMap(m map[string]any, prefix string, result map[string]any) {
	for key, value := range m {
		// Create the new key
		newKey := key
		if prefix != "" {
			newKey = prefix + "_" + key
		}

		// Check if value is a nested map
		if nestedMap, ok := value.(map[string]any); ok {
			// Recursively flatten nested maps
			pe.flattenMap(nestedMap, newKey, result)
		} else {
			// Add the leaf value
			result[newKey] = value
		}
	}
}

// evaluateExpression evaluates a simple expression using the expression evaluator
func (pe *PropertyEngine) evaluateExpression(resource map[string]any, expression string) (any, error) {
	return pe.evaluator.Evaluate(expression, resource)
}
