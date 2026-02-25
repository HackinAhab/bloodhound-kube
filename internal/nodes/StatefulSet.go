package nodes

type StatefulSetCore struct {
	GraphNodeBase
	SelectorMap map[string]any
}

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
		Namespace: namespace,
		Cluster:   false,
		Data: StatefulSetCore{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("StatefulSet", namespace, name),
				Kinds:          []string{"StatefulSet"},
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
			ID:         BuildID("StatefulSet", namespace, name),
			Kinds:      []string{"StatefulSet"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
