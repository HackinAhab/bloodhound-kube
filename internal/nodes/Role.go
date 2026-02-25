package nodes

func init() {
	Register("Role", BuildRoleNode)
}

type Role struct {
	GraphNodeBase
	PermsDisplay []string
	RbacRules    []RbacRule
}

func BuildRoleNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")
	parsedRBACRules := buildRbacRules(GetSlice(resource, "rules"))

	perms := buildRbacRulesDisplay(parsedRBACRules)
	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"perms":       perms,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Role{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("Role", namespace, name),
				Kinds:          []string{"Role"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			PermsDisplay: perms,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Role", namespace, name),
			Kinds:      []string{"Role"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
