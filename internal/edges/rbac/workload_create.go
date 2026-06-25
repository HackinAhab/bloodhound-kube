package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type rbacCreateEdgesRule struct{}

func (r rbacCreateEdgesRule) Name() string { return "rbac_create" }

var edgePropertiesRBACCreate = map[string]any{
	"Description": "Identity has RBAC permissions to create RoleBindings or ClusterRoleBindings",
	"Reference":   "",
}

func (r rbacCreateEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, rbacCreateNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, rbacCreateCluster(ctx)...)
	return edges
}

func rbacCreateNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"rolebindings", "rbac.authorization.k8s.io/rolebindings"}
	verbs := []string{"create"}

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
			if len(space.AllRoles) > 0 {
				agg := &space.AllRoles[0]
				edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_RBACCreate", edgePropertiesRBACCreate))
			}
			if len(ctx.Core.Cluster.AllClusterRoles) > 0 {
				agg := &ctx.Core.Cluster.AllClusterRoles[0]
				edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_RBACCreate", edgePropertiesRBACCreate))
			}
		}
	}
	return edges
}

func rbacCreateCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"clusterrolebindings", "rbac.authorization.k8s.io/clusterrolebindings"}
	verbs := []string{"create"}

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
			if len(ctx.Core.Cluster.AllClusterRoles) > 0 {
				agg := &ctx.Core.Cluster.AllClusterRoles[0]
				edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_RBACCreate", edgePropertiesRBACCreate))
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				if len(space.AllRoles) > 0 {
					agg := &space.AllRoles[0]
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_RBACCreate", edgePropertiesRBACCreate))
				}
			}
		}
	}
	return edges
}

type rbacCreateWorkloadEdgesRule struct{}

func (r rbacCreateWorkloadEdgesRule) Name() string { return "rbac_create_workload" }

var edgePropertiesRBACWorkloadCreate = map[string]any{
	"Description": "Identity has RBAC permissions to create workloads",
	"Reference":   "https://kubernetes.io/docs/reference/access-authn-authz/rbac/#referring-to-resources",
}

func (r rbacCreateWorkloadEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, workloadCreateNamespaced(ctx, ns)...)
	}
	edges = append(edges, workloadCreateCluster(ctx)...)
	return edges
}

func workloadCreateNamespaced(ctx *framework.Context, namespace string) []model.BloodHoundEdge {
	if ctx == nil {
		return nil
	}
	resourceKeys := []string{"pods", "deployments", "apps/deployments", "daemonsets", "apps/daemonsets", "statefulsets", "apps/statefulsets", "jobs", "batch/jobs", "cronjobs", "batch/cronjobs", "replicationcontrollers"}
	verbs := []string{"create"}

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
			for i := range ctx.Core.Cluster.Nodes {
				node := &ctx.Core.Cluster.Nodes[i]
				edges = append(edges, framework.CreateEdgeWithProperties(principal, node, "BHK_WorkloadCreate", edgePropertiesRBACWorkloadCreate))
			}
		}
	}
	return edges
}

func workloadCreateCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"pods", "deployments", "apps/deployments", "daemonsets", "apps/daemonsets", "statefulsets", "apps/statefulsets", "jobs", "batch/jobs", "cronjobs", "batch/cronjobs", "replicationcontrollers"}
	verbs := []string{"create"}

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
			for i := range ctx.Core.Cluster.Nodes {
				node := &ctx.Core.Cluster.Nodes[i]
				edges = append(edges, framework.CreateEdgeWithProperties(principal, node, "BHK_WorkloadCreate", edgePropertiesRBACWorkloadCreate))
			}
		}
	}
	return edges
}
