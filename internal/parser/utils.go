package parser

import (
	"fmt"
	"maps"
	"strings"
)

// FlattenProperties converts nested objects to primitive key-value pairs
func FlattenProperties(input map[string]any, prefix string) map[string]any {
	result := make(map[string]any)

	if input == nil {
		return result
	}

	for key, value := range input {
		flatKey := key
		if prefix != "" {
			flatKey = prefix + "_" + key
		}

		switch v := value.(type) {
		case map[string]any:
			// Recursively flatten nested objects
			nested := FlattenProperties(v, flatKey)
			maps.Copy(result, nested)
		case []any:
			// Convert object arrays to primitive arrays or flatten if needed
			if len(v) > 0 {
				if isObjectArray(v) {
					result[flatKey] = convertObjectArrayToPrimitives(v)
				} else {
					result[flatKey] = v
				}
			}
		default:
			result[flatKey] = value
		}
	}

	return result
}

// SanitizeProperties ensures properties conform to the ingest schema.
// It removes nulls, flattens nested objects, and normalizes arrays to primitives.
func SanitizeProperties(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}

	flattened := FlattenProperties(input, "")
	if len(flattened) == 0 {
		return nil
	}

	result := make(map[string]any)
	for key, value := range flattened {
		// Preserve label and annotation maps as they are used for edge generation
		// This is a bit of a hack, but it allows us to keep the original maps for edge generation while still flattening other properties.
		if key == "__private" || strings.Contains(key, "__private_") {
			continue
		}
		if value == nil {
			continue
		}

		switch v := value.(type) {
		case map[string]any:
			// FlattenProperties should have removed maps, but skip just in case.
			continue
		case []any:
			normalized := normalizePrimitiveArray(v)
			if len(normalized) > 0 {
				result[key] = normalized
			}
		case []string, []bool, []int, []int64, []float64, []float32:
			normalized := normalizeTypedArray(v)
			if len(normalized) > 0 {
				result[key] = normalized
			}
		default:
			if isPrimitive(value) {
				result[key] = value
			}
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func normalizeTypedArray(value any) []any {
	switch v := value.(type) {
	case []string:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	case []bool:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	case []int:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	case []int64:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	case []float64:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	case []float32:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func normalizePrimitiveArray(values []any) []any {
	if len(values) == 0 {
		return nil
	}

	primitiveKind := ""
	for _, item := range values {
		if item == nil {
			continue
		}
		kind := primitiveKindOf(item)
		if kind == "" {
			return stringifyArray(values)
		}
		if primitiveKind == "" {
			primitiveKind = kind
		} else if primitiveKind != kind {
			return stringifyArray(values)
		}
	}

	out := make([]any, 0, len(values))
	for _, item := range values {
		if item == nil {
			continue
		}
		if isPrimitive(item) {
			out = append(out, item)
		}
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func stringifyArray(values []any) []any {
	out := make([]any, 0, len(values))
	for _, item := range values {
		if item == nil {
			continue
		}
		out = append(out, fmt.Sprint(item))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isPrimitive(value any) bool {
	return primitiveKindOf(value) != ""
}

func primitiveKindOf(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case int, int8, int16, int32, int64:
		return "number"
	case uint, uint8, uint16, uint32, uint64:
		return "number"
	case float32, float64:
		return "number"
	default:
		return ""
	}
}

// isObjectArray checks if array contains objects
func isObjectArray(arr []any) bool {
	if len(arr) == 0 {
		return false
	}
	_, isObj := arr[0].(map[string]any)
	return isObj
}

// convertObjectArrayToPrimitives converts object arrays to string arrays
func convertObjectArrayToPrimitives(arr []any) []string {
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if obj, ok := item.(map[string]any); ok {
			if name, exists := obj["name"]; exists {
				if nameStr, ok := name.(string); ok {
					result = append(result, nameStr)
				}
			} else if id, exists := obj["id"]; exists {
				if idStr, ok := id.(string); ok {
					result = append(result, idStr)
				}
			}
		}
	}
	return result
}
