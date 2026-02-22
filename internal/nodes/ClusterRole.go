package nodes

func init() {
	Register("ClusterRole", BuildClusterRoleNode)
}

func BuildClusterRoleNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	perms := buildRBACPerms(GetSlice(resource, "rules"))

	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"perms":       perms,
	}

	core := CoreEntry{
		Cluster: true,
		Data: ClusterRoleCore{
			CoreNode: CoreNode{
				ID:             BuildID("ClusterRole", "", name),
				Kinds:          []string{"ClusterRole"},
				Name:           name,
				Namespace:      "",
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			Perms: perms,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("ClusterRole", "", name),
			Kinds:      []string{"ClusterRole"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
