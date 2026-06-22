package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type rbacReadLogsEdgesRule struct{}

func (r rbacReadLogsEdgesRule) Name() string { return "rbac_read_logs" }

var edgePropertiesRBACReadLogs = map[string]any{
	"Description": "Identity has RBAC permissions to read pod logs.",
}

func (r rbacReadLogsEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, readLogsNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, readLogsCluster(ctx)...)
	return edges
}

func readLogsNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"pods/log"}
	verbs := []string{"get", "list", "watch"}

	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	var edges []model.BloodHoundEdge
	for _, binding := range roleBindings {
		perms := permsForBinding(ctx, namespace, binding.RoleKind, binding.RoleName)
		if len(perms) == 0 {
			continue
		}
		all, names := accessForResource(perms, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveNamespacedSubject(ctx, namespace, binding.Namespace, subject)
			if principal == nil {
				continue
			}
			for i := range space.Pods {
				pod := &space.Pods[i]
				if all {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, pod, "BHK_ReadLogs", edgePropertiesRBACReadLogs))
					continue
				}
				if _, ok := names[pod.Name]; ok {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, pod, "BHK_ReadLogs", edgePropertiesRBACReadLogs))
				}
			}
		}
	}
	return edges
}

func readLogsCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"pods/log"}
	verbs := []string{"get", "list", "watch"}

	var edges []model.BloodHoundEdge
	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		all, names := accessForResource(clusterRole.PermsDisplay, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveClusterSubject(ctx, subject)
			if principal == nil {
				continue
			}
			if all {
				if len(ctx.Core.Cluster.AllPods) > 0 {
					agg := &ctx.Core.Cluster.AllPods[0]
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_ReadLogs", edgePropertiesRBACReadLogs))
				}
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for i := range space.Pods {
					pod := &space.Pods[i]
					if _, ok := names[pod.Name]; ok {
						edges = append(edges, framework.CreateEdgeWithProperties(principal, pod, "BHK_ReadLogs", edgePropertiesRBACReadLogs))
					}
				}
			}
		}
	}
	return edges
}
