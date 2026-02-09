package parser

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"sort"
	"strings"
)

// GenerateObjectID creates a unique identifier for a Kubernetes resource
func GenerateObjectID(resourceType, namespace, name string) string {
	var identifier string
	if namespace != "" {
		identifier = fmt.Sprintf("%s/%s/%s", resourceType, namespace, name)
	} else {
		identifier = fmt.Sprintf("%s/%s", resourceType, name)
	}

	hash := sha256.Sum256([]byte(identifier))
	return fmt.Sprintf("%x", hash)[:16]
}

// GenerateNodeID creates a unique node identifier for BloodHound
func GenerateNodeID(label, resourceType, namespace, name string) string {
	baseID := GenerateObjectID(resourceType, namespace, name)
	return fmt.Sprintf("%s:%s", label, baseID)
}

// SanitizeLabel sanitizes a label for BloodHound compatibility
func SanitizeLabel(label string) string {
	sanitized := strings.ToUpper(label)
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	return sanitized
}

// SanitizePascalCase converts edge kinds to PascalCase without dashes
func SanitizePascalCase(input string) string {
	words := strings.FieldsFunc(input, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})

	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]) + strings.ToLower(word[1:]))
		}
	}
	return result.String()
}

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
		if key == "labels_map" || key == "annotations_map" || strings.HasPrefix(key, "labels_map_") || strings.HasPrefix(key, "annotations_map_") {
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

// DeduplicateEdges removes duplicate edges based on start→end:kind key
func DeduplicateEdges(edges []BloodHoundEdge) []BloodHoundEdge {
	seen := make(map[string]bool)
	var unique []BloodHoundEdge

	for _, edge := range edges {
		key := fmt.Sprintf("%s→%s:%s", edge.Start.Value, edge.End.Value, edge.Kind)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, edge)
		}
	}

	return unique
}

// SortEdgesByKind sorts edges alphabetically by kind
func SortEdgesByKind(edges []BloodHoundEdge) {
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Kind < edges[j].Kind
	})
}

// GetEdgeStats returns statistics about edges
func GetEdgeStats(edges []BloodHoundEdge) map[string]int {
	stats := make(map[string]int)

	for _, edge := range edges {
		stats[edge.Kind]++
	}

	return stats
}
