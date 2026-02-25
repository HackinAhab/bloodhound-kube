package nodes

type ConfigMap struct {
	GraphNodeBase
	Data map[string]any
}

func init() {
	Register("ConfigMap", BuildConfigMapNode)
}

func BuildConfigMapNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	data := GetMap(resource, "data")
	keys := MapKeysSorted(data)
	entries := MapEntriesSorted(data)

	properties := map[string]any{
		"name":          name,
		"namespace":     namespace,
		"labels":        MapToSortedList(labelsMap),
		"annotations":   MapToSortedList(annotationsMap),
		"dataKeys":      keys,
		"dataKeysCount": len(keys),
		"dataEntries":   entries,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: ConfigMap{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("ConfigMap", namespace, name),
				Kinds:          []string{"ConfigMap"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			Data: data,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("ConfigMap", namespace, name),
			Kinds:      []string{"ConfigMap"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
