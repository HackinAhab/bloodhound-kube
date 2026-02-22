package nodes

func init() {
	Register("RoleBinding", BuildRoleBindingNode)
}

func BuildRoleBindingNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	subjects := GetSlice(resource, "subjects")
	subjectCores := extractRbacSubjectCores(subjects)
	roleRef := GetMap(resource, "roleRef")
	roleName := GetString(roleRef, "name")
	roleKind := GetString(roleRef, "kind")

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"roleName":    roleName,
		"roleKind":    roleKind,
		"subjects":    summarizeRbacSubjects(subjects, namespace),
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: RoleBindingCore{
			CoreNode: CoreNode{
				ID:             BuildID("RoleBinding", namespace, name),
				Kinds:          []string{"RoleBinding"},
				Name:           name,
				Namespace:      namespace,
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
			ID:         BuildID("RoleBinding", namespace, name),
			Kinds:      []string{"RoleBinding"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
