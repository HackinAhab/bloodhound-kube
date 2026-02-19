package nodes

func init() {
	Register("", BuildGenericNode)
}

func BuildGenericNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}

	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	kind := GetString(resource, "kind")
	id := BuildID(kind, namespace, name)

	return BuildResult{
		Node: NodeResult{
			ID:         id,
			Kinds:      []string{kind},
			Properties: properties,
		},
	}, true
}
