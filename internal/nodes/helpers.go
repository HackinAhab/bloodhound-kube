package nodes

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// BuildID creates a stable identifier from kind, namespace, and name.
// Example: BuildID("Pod", "default", "nginx") -> "Pod:default:nginx".
func BuildID(kind, namespace, name string) string {
	if namespace == "" {
		return fmt.Sprintf("%s:%s", kind, name)
	}
	return fmt.Sprintf("%s:%s:%s", kind, namespace, name)
}

// GetMap returns a map value for key or an empty map.
// Example: GetMap(map[string]any{"meta": map[string]any{"a": 1}}, "meta") -> map[a:1].
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

// GetSlice returns a slice value for key or an empty slice.
// Example: GetSlice(map[string]any{"items": []any{"a", "b"}}, "items") -> []any{"a", "b"}.
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

// GetString returns a string value for key or empty string.
// Example: GetString(map[string]any{"name": "kube"}, "name") -> "kube".
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

// GetStringDefault returns a string value or a fallback.
// Example: GetStringDefault(map[string]any{}, "name", "unknown") -> "unknown".
func GetStringDefault(parent map[string]any, key, fallback string) string {
	value := GetString(parent, key)
	if value == "" {
		return fallback
	}
	return value
}

// GetBool returns a boolean-ish value for key or false.
// Example: GetBool(map[string]any{"ready": true}, "ready") -> true.
func GetBool(parent map[string]any, key string) any {
	if parent == nil {
		return false
	}
	value, ok := parent[key]
	if !ok || value == nil {
		return false
	}
	return value
}

// GetBoolValue returns a boolean value for key or false.
// It normalizes common boolean-like values.
func GetBoolValue(parent map[string]any, key string) bool {
	if parent == nil {
		return false
	}
	value, ok := parent[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

// GetNumber returns an int value for key or 0.
// Example: GetNumber(map[string]any{"replicas": float64(3)}, "replicas") -> 3.
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

// MapToSortedList formats a map into sorted "key=value" entries.
// Example: MapToSortedList(map[string]any{"b": 2, "a": 1}) -> []string{"a=1", "b=2"}.
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

// MapKeysSorted returns sorted keys for a map.
// Example: MapKeysSorted(map[string]any{"b": 2, "a": 1}) -> []string{"a", "b"}.
func MapKeysSorted(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// MapEntriesSorted formats a map into sorted "key=value" entries.
// Example: MapEntriesSorted(map[string]any{"b": 2, "a": 1}) -> []string{"a=1", "b=2"}.
func MapEntriesSorted(m map[string]any) []string {
	keys := MapKeysSorted(m)
	entries := make([]string, 0, len(keys))
	for _, k := range keys {
		entries = append(entries, fmt.Sprintf("%s=%v", k, m[k]))
	}
	return entries
}

// SortedSetKeys returns sorted keys from a set.
// Example: SortedSetKeys(map[string]struct{}{"b": {}, "a": {}}) -> []string{"a", "b"}.
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

// StringSlice extracts string values from a mixed slice.
// Example: StringSlice([]any{"a", 1, "b"}) -> []string{"a", "b"}.
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

// SeLinuxSummary formats an SELinux options map into a summary string.
// Example: SeLinuxSummary(map[string]any{"user": "u", "role": "r", "type": "t", "level": "l"}) -> "user=u, role=r, type=t, level=l".
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

// AppArmorProfileValue extracts the profile value from security context.
// Example: AppArmorProfileValue(map[string]any{"appArmorProfile": map[string]any{"type": "RuntimeDefault"}}) -> "RuntimeDefault".
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

// RemoveKeyFromSlice removes a string from a slice.
// Example: RemoveKeyFromSlice([]string{"a", "b"}, "a") -> []string{"b"}.
func RemoveKeyFromSlice(slice []string, key string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != key {
			result = append(result, item)
		}
	}
	return result
}

func summarizeRbacSubjects(subjects []any, defaultNamespace string) []string {
	if len(subjects) == 0 {
		return []string{}
	}
	entries := make([]string, 0, len(subjects))
	for _, item := range subjects {
		subject, ok := item.(map[string]any)
		if !ok {
			continue
		}
		kind := GetString(subject, "kind")
		name := GetString(subject, "name")
		namespace := GetString(subject, "namespace")
		if namespace == "" {
			namespace = defaultNamespace
		}
		if kind == "" || name == "" {
			continue
		}
		if namespace != "" {
			entries = append(entries, kind+":"+namespace+"/"+name)
			continue
		}
		entries = append(entries, kind+":"+name)
	}
	sort.Strings(entries)
	return entries
}

func extractRbacSubjectCores(subjects []any) []SubjectCore {
	if len(subjects) == 0 {
		return []SubjectCore{}
	}
	entries := make([]SubjectCore, 0, len(subjects))
	for _, item := range subjects {
		subject, ok := item.(map[string]any)
		if !ok {
			continue
		}
		kind := GetString(subject, "kind")
		name := GetString(subject, "name")
		if kind == "" || name == "" {
			continue
		}
		entries = append(entries, SubjectCore{
			Kind:      kind,
			Name:      name,
			Namespace: GetString(subject, "namespace"),
		})
	}
	return entries
}

func buildRBACPerms(rules []any) []string {
	resourceMap := map[string]map[string]struct{}{}
	nonResourceMap := map[string]map[string]struct{}{}

	for _, item := range rules {
		rule, ok := item.(map[string]any)
		if !ok {
			continue
		}
		verbs := StringSlice(GetSlice(rule, "verbs"))
		if len(verbs) == 0 {
			continue
		}
		addVerbs := func(target map[string]map[string]struct{}, key string) {
			if key == "" {
				return
			}
			if target[key] == nil {
				target[key] = map[string]struct{}{}
			}
			for _, verb := range verbs {
				target[key][verb] = struct{}{}
			}
		}

		for _, url := range StringSlice(GetSlice(rule, "nonResourceURLs")) {
			addVerbs(nonResourceMap, url)
		}

		resources := StringSlice(GetSlice(rule, "resources"))
		if len(resources) == 0 {
			continue
		}
		apiGroups := StringSlice(GetSlice(rule, "apiGroups"))
		if len(apiGroups) == 0 {
			apiGroups = []string{""}
		}
		resourceNames := StringSlice(GetSlice(rule, "resourceNames"))
		if len(resourceNames) == 0 {
			resourceNames = []string{""}
		}

		for _, resource := range resources {
			for _, group := range apiGroups {
				for _, name := range resourceNames {
					key := buildResourceKey(group, resource, name)
					addVerbs(resourceMap, key)
				}
			}
		}
	}

	keys := make([]string, 0, len(resourceMap)+len(nonResourceMap))
	for key := range resourceMap {
		keys = append(keys, key)
	}
	for key := range nonResourceMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	perms := make([]string, 0, len(keys))
	for _, key := range keys {
		verbs := make([]string, 0)
		if set, ok := resourceMap[key]; ok {
			verbs = append(verbs, SortedSetKeys(set)...)
		}
		if set, ok := nonResourceMap[key]; ok {
			verbs = append(verbs, SortedSetKeys(set)...)
		}
		sort.Strings(verbs)
		if len(verbs) == 0 {
			continue
		}
		perms = append(perms, key+": "+joinWithComma(verbs))
	}

	return perms
}

func buildResourceKey(group, resource, name string) string {
	if name != "" {
		base := resource
		if group != "" {
			base = group + "/" + resource
		}
		return base + "/" + name
	}
	if group != "" {
		return group + "/" + resource
	}
	return resource
}

func joinWithComma(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, ", ")
}
