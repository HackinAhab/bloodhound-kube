package nodes

import (
	"encoding/json"
	"fmt"
	"sort"
)

func BuildID(kind, namespace, name string) string {
	if namespace == "" {
		return fmt.Sprintf("%s:%s", kind, name)
	}
	return fmt.Sprintf("%s:%s:%s", kind, namespace, name)
}

func GetMap(parent map[string]any, key string) map[string]any {
	if parent == nil {
		return map[string]any{}
	}
	value, ok := parent[key]
	if !ok || value == nil {
		return map[string]any{}
	}
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func GetSlice(parent map[string]any, key string) []any {
	if parent == nil {
		return []any{}
	}
	value, ok := parent[key]
	if !ok || value == nil {
		return []any{}
	}
	if s, ok := value.([]any); ok {
		return s
	}
	return []any{}
}

func GetString(parent map[string]any, key string) string {
	if parent == nil {
		return ""
	}
	value, ok := parent[key]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func GetStringDefault(parent map[string]any, key, fallback string) string {
	value := GetString(parent, key)
	if value == "" {
		return fallback
	}
	return value
}

func GetValue(parent map[string]any, key string) any {
	if parent == nil {
		return ""
	}
	value, ok := parent[key]
	if !ok || value == nil {
		return ""
	}
	return value
}

func GetNumber(parent map[string]any, key string) int {
	if parent == nil {
		return 0
	}
	value, ok := parent[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func MapToSortedList(m map[string]any) []string {
	if len(m) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	items := make([]string, 0, len(keys))
	for _, k := range keys {
		items = append(items, fmt.Sprintf("%s=%v", k, m[k]))
	}
	return items
}

func MapKeysSorted(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func MapEntriesSorted(m map[string]any) []string {
	keys := MapKeysSorted(m)
	entries := make([]string, 0, len(keys))
	for _, k := range keys {
		entries = append(entries, fmt.Sprintf("%s=%v", k, m[k]))
	}
	return entries
}

func SortedSetKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(set))
	for key := range set {
		items = append(items, key)
	}
	sort.Strings(items)
	return items
}

func StringSlice(values []any) []string {
	if len(values) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok {
			items = append(items, s)
		}
	}
	return items
}

func SeLinuxSummary(options map[string]any) string {
	if len(options) == 0 {
		return ""
	}
	user := GetString(options, "user")
	role := GetString(options, "role")
	typ := GetString(options, "type")
	level := GetString(options, "level")
	return fmt.Sprintf("user=%v, role=%v, type=%v, level=%v", user, role, typ, level)
}

func AppArmorProfileValue(sec map[string]any) string {
	value, ok := sec["appArmorProfile"]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	if m, ok := value.(map[string]any); ok {
		if typ, ok := m["type"].(string); ok {
			return typ
		}
		if profile, ok := m["localhostProfile"].(string); ok {
			return profile
		}
		if data, err := json.Marshal(m); err == nil {
			return string(data)
		}
	}
	return ""
}

func RemoveKeyFromSlice(slice []string, key string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != key {
			result = append(result, item)
		}
	}
	return result
}
