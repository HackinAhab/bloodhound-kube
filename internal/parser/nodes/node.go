package nodes

func init() {
	Register("Node", BuildNodeNode)
}

func BuildNodeNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")
	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	core := CoreEntry{
		Key:     "nodes",
		Cluster: true,
		Data: map[string]any{
			"id":              BuildID("Node", "", name),
			"kinds":           []string{"Node"},
			"name":            name,
			"labels_map":      labelsMap,
			"annotations_map": annotationsMap,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Node", "", name),
			Kinds:      []string{"Node"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
