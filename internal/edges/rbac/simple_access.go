package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
)

// simpleAccessRule handles the common "identity has RBAC access to a single
// resource kind" shape shared by read-secret/configmap, pod exec/attach/
// portforward/debug, read-logs, and SA-token-request rules: parse a
// Role/ClusterRole's permissions for resourceKeys+verbs, resolve the binding's
// subjects, and edge to the AllX aggregate when access is unrestricted or to
// each individually-named target otherwise.
//
// Rules with any other shape (dual target kinds, dynamic edge kind,
// multi-resource checks, or "fan out when no aggregate exists" fallbacks) do
// not fit this template — see node_proxy.go, scc_usage.go, impersonate.go,
// and escalate_bind.go.
type simpleAccessRule[T nodefw.EdgeNode, A nodefw.EdgeNode] struct {
	name         string
	resourceKeys []string
	verbs        []string
	edgeKind     string
	props        map[string]any

	namespacedTargets func(*model.Namespace) []T
	namespacedAll     func(*model.Namespace) []A
	clusterAll        func(*model.Cluster) []A
}

func (r simpleAccessRule[T, A]) Name() string { return r.name }

func (r simpleAccessRule[T, A]) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, r.applyNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, r.applyCluster(ctx)...)
	return edges
}

func (r simpleAccessRule[T, A]) applyNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	targets := r.namespacedTargets(space)
	aggregate := r.namespacedAll(space)

	var edges []model.BloodHoundEdge
	for _, binding := range ctx.Index.RoleBindingsByNamespace[namespace] {
		perms := permsForBinding(ctx, namespace, binding.RoleKind, binding.RoleName)
		if len(perms) == 0 {
			continue
		}
		all, names := accessForResource(perms, r.resourceKeys, r.verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveNamespacedSubject(ctx, namespace, binding.Namespace, subject)
			if principal == nil {
				continue
			}
			if all {
				if agg := framework.FirstEdgeNode(aggregate); agg != nil {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, r.edgeKind, r.props))
				}
				continue
			}
			for i := range targets {
				if _, ok := names[targets[i].EdgeName()]; ok {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, targets[i], r.edgeKind, r.props))
				}
			}
		}
	}
	return edges
}

func (r simpleAccessRule[T, A]) applyCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx.Core.Cluster == nil {
		return nil
	}
	clusterAggregate := r.clusterAll(ctx.Core.Cluster)

	var edges []model.BloodHoundEdge
	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		all, names := accessForResource(clusterRole.PermsDisplay, r.resourceKeys, r.verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveClusterSubject(ctx, subject)
			if principal == nil {
				continue
			}
			if all {
				if agg := framework.FirstEdgeNode(clusterAggregate); agg != nil {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, r.edgeKind, r.props))
				}
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				targets := r.namespacedTargets(space)
				for i := range targets {
					if _, ok := names[targets[i].EdgeName()]; ok {
						edges = append(edges, framework.CreateEdgeWithProperties(principal, targets[i], r.edgeKind, r.props))
					}
				}
			}
		}
	}
	return edges
}
