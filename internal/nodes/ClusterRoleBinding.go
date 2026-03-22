package nodes

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding"), BuildClusterRoleBindingNode)
}

type ClusterRoleBinding struct {
	GraphNodeBase
	RoleName string
	RoleKind string
	Subjects []Subject
}

func BuildClusterRoleBindingNode(obj runtime.Object) (BuildResult, bool) {
	binding, ok := obj.(*rbacv1.ClusterRoleBinding)
	if !ok || binding == nil {
		return BuildResult{}, false
	}
	name := binding.Name
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := StringMapToAnyMap(binding.Labels)
	annotationsMap := StringMapToAnyMap(binding.Annotations)

	subjectCores := extractRbacSubjectCores(binding.Subjects)
	roleName := binding.RoleRef.Name
	roleKind := binding.RoleRef.Kind

	properties := map[string]any{
		"name":        name,
		"namespace":   "",
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"roleName":    roleName,
		"roleKind":    roleKind,
		"subjects":    summarizeRbacSubjects(binding.Subjects, ""),
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
