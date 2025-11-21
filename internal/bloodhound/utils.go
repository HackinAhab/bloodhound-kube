package bloodhound

import (
	"crypto/sha256"
	"fmt"
	"maps"
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
			// Extract a meaningful string representation
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
