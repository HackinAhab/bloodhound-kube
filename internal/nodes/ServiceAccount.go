package nodes

type ServiceAccount struct {
	GraphNodeBase
	Secrets []string
}

func init() {
	Register("ServiceAccount", BuildServiceAccountNode)
}

func BuildServiceAccountNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	secrets := []string{}
	for _, s := range GetSlice(resource, "secrets") {
		if m, ok := s.(map[string]any); ok {
			if n, ok := m["name"].(string); ok {
				secrets = append(secrets, n)
			}
		}
	}

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"secrets":     secrets,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: ServiceAccount{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("ServiceAccount", namespace, name),
				Kinds:          []string{"ServiceAccount"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			Secrets: secrets,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("ServiceAccount", namespace, name),
			Kinds:      []string{"ServiceAccount"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
