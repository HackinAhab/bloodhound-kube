package nodes

func init() {
	Register("StatefulSet", BuildStatefulSetNode)
}

func BuildStatefulSetNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	selectorMap := GetMap(GetMap(spec, "selector"), "matchLabels")

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"selector":    MapToSortedList(selectorMap),
		"serviceName": GetString(spec, "serviceName"),
	}

	core := CoreEntry{
		Key:       "statefulsets",
		Namespace: namespace,
		Cluster:   false,
		Data: map[string]any{
			"id":              BuildID("StatefulSet", namespace, name),
			"kinds":           []string{"StatefulSet"},
			"name":            name,
			"namespace":       namespace,
			"selector_map":    selectorMap,
			"labels_map":      labelsMap,
			"annotations_map": annotationsMap,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("StatefulSet", namespace, name),
			Kinds:      []string{"StatefulSet"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
