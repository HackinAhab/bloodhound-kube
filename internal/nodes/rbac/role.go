package rbac

import (
	. "bloodhound-kube/internal/nodes/framework"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Role struct {
	GraphNodeBase
	PermsDisplay []string
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

	parsedRBACRules := BuildRbacRules(role.Rules)
	perms := BuildRbacRulesDisplay(parsedRBACRules)

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"perms":       perms,
	}

	base := NewGraphNodeBase("Role", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Role{
			GraphNodeBase: base,
			PermsDisplay:  perms,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}
