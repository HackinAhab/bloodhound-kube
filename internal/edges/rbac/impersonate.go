package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type rbacImpersonateEdgesRule struct{}

func (r rbacImpersonateEdgesRule) Name() string { return "rbac_impersonate" }

var edgePropertiesImpersonate = map[string]any{
	"Description": "Identity has RBAC permissions to impersonate another identity.",
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
		edges = append(edges, impersonateNamespaced(ctx, ns)...)
	}
	edges = append(edges, impersonateCluster(ctx)...)
	return edges
}

func impersonateNamespaced(ctx *framework.Context, namespace string) []model.BloodHoundEdge {
	if ctx == nil {
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
		allSA, saNames := accessForParsedResource(parsed, saResourceKeys, verbs)
		allUsers, userNames := accessForParsedResource(parsed, userResourceKeys, verbs)
		allGroups, groupNames := accessForParsedResource(parsed, groupResourceKeys, verbs)
		if !allSA && len(saNames) == 0 && !allUsers && len(userNames) == 0 && !allGroups && len(groupNames) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveNamespacedSubject(ctx, namespace, binding.Namespace, subject)
			if principal == nil {
				continue
			}

			if allSA || len(saNames) > 0 {
				space := ctx.Core.Namespaces[namespace]
				if space != nil {
					for i := range space.ServiceAccounts {
						target := &space.ServiceAccounts[i]
						if allSA {
							edges = append(edges, framework.CreateEdgeWithProperties(principal, target, "BHK_Impersonate", edgePropertiesImpersonate))
						} else if _, ok := saNames[target.Name]; ok {
							edges = append(edges, framework.CreateEdgeWithProperties(principal, target, "BHK_Impersonate", edgePropertiesImpersonate))
						}
					}
				}
			}

			if allUsers {
				if len(ctx.Core.Cluster.AllUsers) > 0 {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, &ctx.Core.Cluster.AllUsers[0], "BHK_Impersonate", edgePropertiesImpersonate))
				}
			} else if len(userNames) > 0 {
				for name := range userNames {
					user := ctx.Index.UsersByName[name]
					if user != nil {
						edges = append(edges, framework.CreateEdgeWithProperties(principal, user, "BHK_Impersonate", edgePropertiesImpersonate))
					}
				}
			}

			if allGroups {
				if len(ctx.Core.Cluster.AllGroups) > 0 {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, &ctx.Core.Cluster.AllGroups[0], "BHK_Impersonate", edgePropertiesImpersonate))
				}
			} else if len(groupNames) > 0 {
				for name := range groupNames {
					group := ctx.Index.GroupsByName[name]
					if group != nil {
						edges = append(edges, framework.CreateEdgeWithProperties(principal, group, "BHK_Impersonate", edgePropertiesImpersonate))
					}
				}
			}
		}
	}
	return edges
}

func impersonateCluster(ctx *framework.Context) []model.BloodHoundEdge {
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
		allSA, saNames := accessForParsedResource(parsed, saResourceKeys, verbs)
		allUsers, userNames := accessForParsedResource(parsed, userResourceKeys, verbs)
		allGroups, groupNames := accessForParsedResource(parsed, groupResourceKeys, verbs)
		if !allSA && len(saNames) == 0 && !allUsers && len(userNames) == 0 && !allGroups && len(groupNames) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			principal := resolveClusterSubject(ctx, subject)
			if principal == nil {
				continue
			}

			if allSA {
				if len(ctx.Core.Cluster.AllServiceAccounts) > 0 {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, &ctx.Core.Cluster.AllServiceAccounts[0], "BHK_Impersonate", edgePropertiesImpersonate))
				}
			} else if len(saNames) > 0 {
				for _, space := range ctx.Core.Namespaces {
					if space == nil {
						continue
					}
					for i := range space.ServiceAccounts {
						target := &space.ServiceAccounts[i]
						if _, ok := saNames[target.Name]; ok {
							edges = append(edges, framework.CreateEdgeWithProperties(principal, target, "BHK_Impersonate", edgePropertiesImpersonate))
						}
					}
				}
			}

			if allUsers {
				if len(ctx.Core.Cluster.AllUsers) > 0 {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, &ctx.Core.Cluster.AllUsers[0], "BHK_Impersonate", edgePropertiesImpersonate))
				}
			} else if len(userNames) > 0 {
				for name := range userNames {
					user := ctx.Index.UsersByName[name]
					if user != nil {
						edges = append(edges, framework.CreateEdgeWithProperties(principal, user, "BHK_Impersonate", edgePropertiesImpersonate))
					}
				}
			}

			if allGroups {
				if len(ctx.Core.Cluster.AllGroups) > 0 {
					edges = append(edges, framework.CreateEdgeWithProperties(principal, &ctx.Core.Cluster.AllGroups[0], "BHK_Impersonate", edgePropertiesImpersonate))
				}
			} else if len(groupNames) > 0 {
				for name := range groupNames {
					group := ctx.Index.GroupsByName[name]
					if group != nil {
						edges = append(edges, framework.CreateEdgeWithProperties(principal, group, "BHK_Impersonate", edgePropertiesImpersonate))
					}
				}
			}
		}
	}
	return edges
}
