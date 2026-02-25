package nodes

type NetworkPolicy struct {
	GraphNodeBase
	PodSelector map[string]any
}

func init() {
	Register("NetworkPolicy", BuildNetworkPolicyNode)
}

func BuildNetworkPolicyNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	podSelector := GetMap(spec, "podSelector")

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: NetworkPolicy{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("NetworkPolicy", namespace, name),
				Kinds:          []string{"NetworkPolicy"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			PodSelector: podSelector,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("NetworkPolicy", namespace, name),
			Kinds:      []string{"NetworkPolicy"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
