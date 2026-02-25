package nodes

type HTTPRoute struct {
	GraphNodeBase
	BackendRefKeys []string
}

func init() {
	Register("HTTPRoute", BuildHTTPRouteNode)
}

func BuildHTTPRouteNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	backendRefKeys := extractHTTPRouteBackendRefKeys(spec)
	parentRefs := summarizeHTTPRouteParents(GetSlice(spec, "parentRefs"), namespace)
	properties := map[string]any{
		"name":           name,
		"namespace":      namespace,
		"labels":         MapToSortedList(labelsMap),
		"annotations":    MapToSortedList(annotationsMap),
		"hostnames":      StringSlice(GetSlice(spec, "hostnames")),
		"parentRefs":     parentRefs,
		"backendRefKeys": backendRefKeys,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: HTTPRoute{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("HTTPRoute", namespace, name),
				Kinds:          []string{"HTTPRoute"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			BackendRefKeys: backendRefKeys,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("HTTPRoute", namespace, name),
			Kinds:      []string{"HTTPRoute"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func extractHTTPRouteBackendRefKeys(spec map[string]any) []string {
	rules := GetSlice(spec, "rules")
	if len(rules) == 0 {
		return []string{}
	}
	keys := map[string]struct{}{}
	for _, item := range rules {
		rule, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for _, backendItem := range GetSlice(rule, "backendRefs") {
			backend, ok := backendItem.(map[string]any)
			if !ok {
				continue
			}
			if kind := GetString(backend, "kind"); kind != "" && kind != "Service" {
				continue
			}
			name := GetString(backend, "name")
			if name == "" {
				continue
			}
			ns := GetString(backend, "namespace")
			keys[ns+"/"+name] = struct{}{}
		}
	}
	return SortedSetKeys(keys)
}

func summarizeHTTPRouteParents(parents []any, defaultNamespace string) []string {
	if len(parents) == 0 {
		return []string{}
	}
	entries := make([]string, 0, len(parents))
	for _, item := range parents {
		parent, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := GetString(parent, "name")
		if name == "" {
			continue
		}
		kind := GetString(parent, "kind")
		if kind == "" {
			kind = "Gateway"
		}
		ns := GetString(parent, "namespace")
		if ns == "" {
			ns = defaultNamespace
		}
		if ns != "" {
			entries = append(entries, kind+":"+ns+"/"+name)
			continue
		}
		entries = append(entries, kind+":"+name)
	}
	return entries
}
