package nodes

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("ClusterRole"), BuildClusterRoleNode)
}

type ClusterRole struct {
	GraphNodeBase
	PermsDisplay []string
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

	base := NewGraphNodeBase("ClusterRole", "", name, labelsMap, annotationsMap)

	core := CoreEntry{
		Cluster: true,
		Data: ClusterRole{
			GraphNodeBase: base,
			PermsDisplay:  perms,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}
