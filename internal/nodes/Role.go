package nodes

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Role struct {
	GraphNodeBase
	PermsDisplay []string
	RbacRules    []RbacRule
}

func BuildRoleNode(obj runtime.Object) (BuildResult, bool) {
	role, ok := obj.(*rbacv1.Role)
	if !ok || role == nil {
		return BuildResult{}, false
	}
	name := role.Name
	if name == "" {
		return BuildResult{}, false
	}
	namespace := role.Namespace
	labelsMap := StringMapToAnyMap(role.Labels)
	annotationsMap := StringMapToAnyMap(role.Annotations)

	parsedRBACRules := buildRbacRules(role.Rules)
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
