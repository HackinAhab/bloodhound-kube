package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type rbacImpersonateEdgesRule struct{}

func (r rbacImpersonateEdgesRule) Name() string { return "rbac_impersonate" }

var edgePropertiesRBACImpersonate = map[string]any{
	"Description": "ServiceAccount has RBAC permissions to impersonate another ServiceAccount",
	"Reference":   "https://kubehound.io/reference/attacks/IDENTITY_IMPERSONATE/",
}

var edgePropertiesRBACImpersonateUsers = map[string]any{
	"Description": "ServiceAccount has RBAC permissions to impersonate arbitrary users, allowing it to act as any user identity in the cluster.",
	"Reference":   "https://kubernetes.io/docs/reference/access-authn-authz/authentication/#user-impersonation",
}

var edgePropertiesRBACImpersonateGroups = map[string]any{
	"Description": "ServiceAccount has RBAC permissions to impersonate groups, including potentially system:masters which grants full cluster-admin access.",
	"Reference":   "https://kubernetes.io/docs/reference/access-authn-authz/authentication/#user-impersonation",
}

func (r rbacImpersonateEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, saImpersonateNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, saImpersonateCluster(ctx)...)
	return edges
}

func saImpersonateNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	saResourceKeys := []string{"serviceaccounts"}
	userResourceKeys := []string{"users"}
	groupResourceKeys := []string{"groups"}
	verbs := []string{"impersonate"}

	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	var edges []model.BloodHoundEdge
	for _, binding := range roleBindings {
		perms := permsForBinding(ctx, namespace, binding.RoleKind, binding.RoleName)
		if len(perms) == 0 {
			continue
		}
		parsed := parseRBACPerms(perms)
		all, names := accessForParsedResource(parsed, saResourceKeys, verbs)
		canImpersonateUsers, _ := accessForParsedResource(parsed, userResourceKeys, verbs)
		canImpersonateGroups, _ := accessForParsedResource(parsed, groupResourceKeys, verbs)
		if !all && len(names) == 0 && !canImpersonateUsers && !canImpersonateGroups {
			continue
		}
		for _, subject := range binding.Subjects {
			sa := resolveNamespacedSubjectSA(ctx, namespace, binding.Namespace, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			for i := range space.ServiceAccounts {
				target := &space.ServiceAccounts[i]
				if all {
					edges = append(edges, framework.CreateEdgeWithProperties(sa, target, "ImpersonateSA", edgePropertiesRBACImpersonate))
					continue
				}
				if _, ok := names[target.Name]; ok {
					edges = append(edges, framework.CreateEdgeWithProperties(sa, target, "ImpersonateSA", edgePropertiesRBACImpersonate))
				}
			}
			if canImpersonateUsers {
				if len(ctx.Core.Cluster.AllServiceAccounts) > 0 {
					agg := &ctx.Core.Cluster.AllServiceAccounts[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "ImpersonateUsers", edgePropertiesRBACImpersonateUsers))
				}
			}
			if canImpersonateGroups {
				if len(ctx.Core.Cluster.AllServiceAccounts) > 0 {
					agg := &ctx.Core.Cluster.AllServiceAccounts[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "ImpersonateGroups", edgePropertiesRBACImpersonateGroups))
				}
			}
		}
	}
	return edges
}

func saImpersonateCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	saResourceKeys := []string{"serviceaccounts"}
	userResourceKeys := []string{"users"}
	groupResourceKeys := []string{"groups"}
	verbs := []string{"impersonate"}

	var edges []model.BloodHoundEdge
	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		parsed := parseRBACPerms(clusterRole.PermsDisplay)
		all, names := accessForParsedResource(parsed, saResourceKeys, verbs)
		canImpersonateUsers, _ := accessForParsedResource(parsed, userResourceKeys, verbs)
		canImpersonateGroups, _ := accessForParsedResource(parsed, groupResourceKeys, verbs)
		if !all && len(names) == 0 && !canImpersonateUsers && !canImpersonateGroups {
			continue
		}
		for _, subject := range binding.Subjects {
			sa := resolveClusterSubjectSA(ctx, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			if len(ctx.Core.Cluster.AllServiceAccounts) > 0 {
				agg := &ctx.Core.Cluster.AllServiceAccounts[0]
				if all {
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "ImpersonateSA", edgePropertiesRBACImpersonate))
				}
				if canImpersonateUsers {
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "ImpersonateUsers", edgePropertiesRBACImpersonateUsers))
				}
				if canImpersonateGroups {
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "ImpersonateGroups", edgePropertiesRBACImpersonateGroups))
				}
			}
			if !all && len(names) > 0 {
				for _, space := range ctx.Core.Namespaces {
					if space == nil {
						continue
					}
					for i := range space.ServiceAccounts {
						target := &space.ServiceAccounts[i]
						if _, ok := names[target.Name]; ok {
							edges = append(edges, framework.CreateEdgeWithProperties(sa, target, "ImpersonateSA", edgePropertiesRBACImpersonate))
						}
					}
				}
			}
		}
	}
	return edges
}
