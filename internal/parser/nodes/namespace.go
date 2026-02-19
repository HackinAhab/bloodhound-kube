package nodes

func init() {
	Register("Namespace", BuildNamespaceNode)
}

func BuildNamespaceNode(resource map[string]any) (BuildResult, bool) {
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

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Namespace", "", name),
			Kinds:      []string{"Namespace"},
			Properties: properties,
		},
	}, true
}
