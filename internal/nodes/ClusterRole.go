package nodes

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ClusterRole struct {
	GraphNodeBase
	PermsDisplay []string
	RbacRules    []RbacRule
}

func BuildClusterRoleNode(obj runtime.Object) (BuildResult, bool) {
	role, ok := obj.(*rbacv1.ClusterRole)
	if !ok || role == nil {
		return BuildResult{}, false
	}
	name := role.Name
	if name == "" {
		return BuildResult{}, false
	}
	labelsMap := StringMapToAnyMap(role.Labels)
	annotationsMap := StringMapToAnyMap(role.Annotations)

	parsedRBACRules := buildRbacRules(role.Rules)
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
