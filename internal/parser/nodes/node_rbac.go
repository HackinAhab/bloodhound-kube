package nodes

import (
	"sort"
	"strings"
)

func init() {
	Register("Role", BuildRoleNode)
	Register("ClusterRole", BuildClusterRoleNode)
	Register("RoleBinding", BuildRoleBindingNode)
	Register("ClusterRoleBinding", BuildClusterRoleBindingNode)
}

func BuildRoleNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	perms := buildRBACPerms(GetSlice(resource, "rules"))

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"perms":       perms,
	}

	core := CoreEntry{
		Key:       "roles",
		Namespace: namespace,
		Cluster:   false,
		Data: map[string]any{
			"id":              BuildID("Role", namespace, name),
			"kinds":           []string{"Role"},
			"name":            name,
			"namespace":       namespace,
			"labels_map":      labelsMap,
			"annotations_map": annotationsMap,
			"perms":           perms,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Role", namespace, name),
			Kinds:      []string{"Role"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func BuildClusterRoleNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	perms := buildRBACPerms(GetSlice(resource, "rules"))

	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"perms":       perms,
	}

	core := CoreEntry{
		Key:     "clusterroles",
		Cluster: true,
		Data: map[string]any{
			"id":              BuildID("ClusterRole", "", name),
			"kinds":           []string{"ClusterRole"},
			"name":            name,
			"labels_map":      labelsMap,
			"annotations_map": annotationsMap,
			"perms":           perms,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("ClusterRole", "", name),
			Kinds:      []string{"ClusterRole"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func BuildRoleBindingNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	subjects := GetSlice(resource, "subjects")
	roleRef := GetMap(resource, "roleRef")
	roleName := GetString(roleRef, "name")
	roleKind := GetString(roleRef, "kind")

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"roleName":    roleName,
		"roleKind":    roleKind,
		"subjects":    summarizeSubjects(subjects, namespace),
	}

	core := CoreEntry{
		Key:       "rolebindings",
		Namespace: namespace,
		Cluster:   false,
		Data: map[string]any{
			"id":              BuildID("RoleBinding", namespace, name),
			"kinds":           []string{"RoleBinding"},
			"name":            name,
			"namespace":       namespace,
			"labels_map":      labelsMap,
			"annotations_map": annotationsMap,
			"subjects":        subjects,
			"roleRef":         roleRef,
			"roleName":        roleName,
			"roleKind":        roleKind,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("RoleBinding", namespace, name),
			Kinds:      []string{"RoleBinding"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func BuildClusterRoleBindingNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	subjects := GetSlice(resource, "subjects")
	roleRef := GetMap(resource, "roleRef")
	roleName := GetString(roleRef, "name")
	roleKind := GetString(roleRef, "kind")

	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"roleName":    roleName,
		"roleKind":    roleKind,
		"subjects":    summarizeSubjects(subjects, ""),
	}

	core := CoreEntry{
		Key:     "clusterrolebindings",
		Cluster: true,
		Data: map[string]any{
			"id":              BuildID("ClusterRoleBinding", "", name),
			"kinds":           []string{"ClusterRoleBinding"},
			"name":            name,
			"labels_map":      labelsMap,
			"annotations_map": annotationsMap,
			"subjects":        subjects,
			"roleRef":         roleRef,
			"roleName":        roleName,
			"roleKind":        roleKind,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("ClusterRoleBinding", "", name),
			Kinds:      []string{"ClusterRoleBinding"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func summarizeSubjects(subjects []any, defaultNamespace string) []string {
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
