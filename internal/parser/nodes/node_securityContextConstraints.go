package nodes

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
		Key:     "securitycontextconstraints",
		Cluster: true,
		Data: map[string]any{
			"id":              BuildID("SecurityContextConstraints", "", name),
			"kinds":           []string{"SecurityContextConstraints"},
			"name":            name,
			"labels_map":      labelsMap,
			"annotations_map": annotationsMap,
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
