package nodes

type SecurityContextConstraints struct {
	GraphNodeBase
}

func init() {
	Register("SecurityContextConstraints", BuildSecurityContextConstraintsNode)
}

func BuildSecurityContextConstraintsNode(resource map[string]any) (BuildResult, bool) {
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
		Cluster: true,
		Data: SecurityContextConstraints{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("SecurityContextConstraints", "", name),
				Kinds:          []string{"SecurityContextConstraints"},
				Name:           name,
				Namespace:      "",
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("SecurityContextConstraints", "", name),
			Kinds:      []string{"SecurityContextConstraints"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
