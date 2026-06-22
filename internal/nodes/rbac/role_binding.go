package rbac

import (
	. "bloodhound-kube/internal/nodes/framework"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type RoleBinding struct {
	GraphNodeBase
	RoleName string
	RoleKind string
	Subjects []Subject
}

func BuildRoleBindingNode(obj runtime.Object) (BuildResult, bool) {
	binding, ok := obj.(*rbacv1.RoleBinding)
	if !ok || binding == nil {
		return BuildResult{}, false
	}
	name := binding.Name
	if name == "" {
		return BuildResult{}, false
	}
	namespace := binding.Namespace
	labelsMap := StringMapToAnyMap(binding.Labels)
	annotationsMap := StringMapToAnyMap(binding.Annotations)

	subjectCores := ExtractRbacSubjectCores(binding.Subjects)
	roleName := binding.RoleRef.Name
	roleKind := binding.RoleRef.Kind

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"roleName":    roleName,
		"roleKind":    roleKind,
		"subjects":    SummarizeRbacSubjects(binding.Subjects, namespace),
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: RoleBinding{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("BHK_RoleBinding", namespace, name),
				Kinds:          []string{"BHK_RoleBinding"},
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
			ID:         BuildID("BHK_RoleBinding", namespace, name),
			Kinds:      []string{"BHK_RoleBinding"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
