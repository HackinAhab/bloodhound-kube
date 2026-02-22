package nodes

func init() {
	Register("Service", BuildServiceNode)
}

func BuildServiceNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	selectorMap := GetMap(spec, "selector")

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"selector":    MapToSortedList(selectorMap),
		"serviceType": GetString(spec, "type"),
		"clusterIP":   GetString(spec, "clusterIP"),
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: ServiceCore{
			CoreNode: CoreNode{
				ID:             BuildID("Service", namespace, name),
				Kinds:          []string{"Service"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			SelectorMap: selectorMap,
			Ports:       GetSlice(spec, "ports"),
			ServiceType: GetString(spec, "type"),
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Service", namespace, name),
			Kinds:      []string{"Service"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
