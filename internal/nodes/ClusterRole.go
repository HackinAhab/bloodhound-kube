package nodes

func init() {
	Register("ClusterRole", BuildClusterRoleNode)
}

type ClusterRole struct {
	GraphNodeBase
	PermsDisplay []string
	RbacRules    []RbacRule
}

func BuildClusterRoleNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	parsedRBACRules := buildRbacRules(GetSlice(resource, "rules"))

	perms := buildRbacRulesDisplay(parsedRBACRules)

	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"perms":       perms,
	}

	core := CoreEntry{
		Cluster: true,
		Data: ClusterRole{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("ClusterRole", "", name),
				Kinds:          []string{"ClusterRole"},
				Name:           name,
				Namespace:      "",
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			PermsDisplay: perms,
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
