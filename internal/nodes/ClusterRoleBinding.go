package nodes

func init() {
	Register("ClusterRoleBinding", BuildClusterRoleBindingNode)
}

type ClusterRoleBinding struct {
	GraphNodeBase
	RoleName string
	RoleKind string
	Subjects []Subject
}

func BuildClusterRoleBindingNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	subjects := GetSlice(resource, "subjects")
	subjectCores := extractRbacSubjectCores(subjects)
	roleRef := GetMap(resource, "roleRef")
	roleName := GetString(roleRef, "name")
	roleKind := GetString(roleRef, "kind")

	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"roleName":    roleName,
		"roleKind":    roleKind,
		"subjects":    summarizeRbacSubjects(subjects, ""),
	}

	core := CoreEntry{
		Cluster: true,
		Data: ClusterRoleBinding{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("ClusterRoleBinding", "", name),
				Kinds:          []string{"ClusterRoleBinding"},
				Name:           name,
				Namespace:      "",
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			RoleName: roleName,
			RoleKind: roleKind,
			Subjects: subjectCores,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("ClusterRoleBinding", "", name),
			Kinds:      []string{"ClusterRoleBinding"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
