package framework

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

type RbacRule struct {
	APIGroup      string
	Resource      string
	Verbs         []string
	ResourceNames []string
}

type Subject struct {
	Kind      string
	Name      string
	Namespace string
}

// BuildID creates a stable identifier from kind, namespace, and name.
// Example: BuildID("Pod", "default", "nginx") -> "Pod:default:nginx".
func BuildID(kind, namespace, name string) string {
	if namespace == "" {
		return fmt.Sprintf("%s:%s", kind, name)
	}
	return fmt.Sprintf("%s:%s:%s", kind, namespace, name)
}

func NewGraphNodeBase(kind, namespace, name string, labelsMap, annotationsMap map[string]any) GraphNodeBase {
	return GraphNodeBase{
		ID:             BuildID(kind, namespace, name),
		Kinds:          []string{kind},
		Name:           name,
		Namespace:      namespace,
		LabelsMap:      labelsMap,
		AnnotationsMap: annotationsMap,
	}
}

func NewNodeResult(base GraphNodeBase, properties map[string]any) NodeResult {
	return NodeResult{
		ID:         base.ID,
		Kinds:      base.Kinds,
		Properties: properties,
	}
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

// StringSliceFromAny filters a []any down to its string elements. Shared by
// addon builders (calico, cilium) that parse unstructured CRD specs.
func StringSliceFromAny(items []any) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func StringMapToAnyMap(input map[string]string) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func I32(value *int32) int {
	if value == nil {
		return 0
	}
	return int(*value)
}

func B(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func Labels(selector map[string]string) map[string]string {
	if selector == nil {
		return map[string]string{}
	}
	return selector
}

func Props(name, namespace string, labelsMap, annotationsMap map[string]any) map[string]any {
	return map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}
}

// MapKeysSorted returns sorted keys for a map.
// Example: MapKeysSorted(map[string]any{"b": 2, "a": 1}) -> []string{"a", "b"}.
func MapKeysSorted(m map[string]any) []string {
	return slices.Sorted(maps.Keys(m))
}

// SortedSetKeys returns sorted keys from a set.
// Example: SortedSetKeys(map[string]struct{}{"b": {}, "a": {}}) -> []string{"a", "b"}.
func SortedSetKeys(set map[string]struct{}) []string {
	return slices.Sorted(maps.Keys(set))
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

func SeLinuxOptionsToMap(options *corev1.SELinuxOptions) map[string]any {
	if options == nil {
		return map[string]any{}
	}
	return map[string]any{
		"user":  options.User,
		"role":  options.Role,
		"type":  options.Type,
		"level": options.Level,
	}
}

func AppArmorProfileValue(profile *corev1.AppArmorProfile) string {
	if profile == nil {
		return ""
	}
	if profile.Type != "" {
		if profile.Type == corev1.AppArmorProfileTypeLocalhost {
			if profile.LocalhostProfile != nil && *profile.LocalhostProfile != "" {
				return *profile.LocalhostProfile
			}
		}
		return string(profile.Type)
	}
	if profile.LocalhostProfile != nil {
		return *profile.LocalhostProfile
	}
	return ""
}

func BuildRbacRulesDisplay(rules []RbacRule) []string {
	entries := make([]string, 0, len(rules))
	for _, rule := range rules {
		var prefix string
		if rule.APIGroup == "" {
			prefix = rule.Resource
		} else if rule.Resource == "" {
			prefix = rule.APIGroup
		} else {
			prefix = rule.APIGroup + "/" + rule.Resource
		}
		verbStr := strings.Join(rule.Verbs, ", ")
		if len(rule.ResourceNames) == 0 {
			entries = append(entries, prefix+": "+verbStr)
		} else {
			for _, name := range rule.ResourceNames {
				entries = append(entries, prefix+"/"+name+": "+verbStr)
			}
		}
	}
	return entries
}

func BuildRbacRules(rules []rbacv1.PolicyRule) []RbacRule {
	parsedRules := make([]RbacRule, 0, len(rules))
	for _, rule := range rules {
		apiGroup := ""
		if len(rule.APIGroups) > 0 {
			apiGroup = rule.APIGroups[0]
		}
		resourceNames := rule.ResourceNames
		verbs := rule.Verbs
		for _, resource := range rule.Resources {
			parsedRule := RbacRule{
				APIGroup:      apiGroup,
				Resource:      resource,
				ResourceNames: resourceNames,
				Verbs:         verbs,
			}
			parsedRules = append(parsedRules, parsedRule)
		}
	}
	return parsedRules
}

func SummarizeRbacSubjects(subjects []rbacv1.Subject, defaultNamespace string) []string {
	if len(subjects) == 0 {
		return []string{}
	}
	entries := make([]string, 0, len(subjects))
	for _, subject := range subjects {
		kind := subject.Kind
		name := subject.Name
		namespace := subject.Namespace
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

func ExtractRbacSubjectCores(subjects []rbacv1.Subject) []Subject {
	if len(subjects) == 0 {
		return []Subject{}
	}
	entries := make([]Subject, 0, len(subjects))
	for _, subject := range subjects {
		kind := subject.Kind
		name := subject.Name
		if kind == "" || name == "" {
			continue
		}
		entries = append(entries, Subject{
			Kind:      kind,
			Name:      name,
			Namespace: subject.Namespace,
		})
	}
	return entries
}
