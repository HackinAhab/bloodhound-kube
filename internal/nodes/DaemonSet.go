package nodes

func init() {
	Register("DaemonSet", BuildDaemonSetNode)
}

func BuildDaemonSetNode(resource map[string]any) (BuildResult, bool) {
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
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: DaemonSetCore{
			CoreNode: CoreNode{
				ID:             BuildID("DaemonSet", namespace, name),
				Kinds:          []string{"DaemonSet"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			SelectorMap: selectorMap,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("DaemonSet", namespace, name),
			Kinds:      []string{"DaemonSet"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
