package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type rbacSATokenRequestEdgesRule struct{}

func (r rbacSATokenRequestEdgesRule) Name() string { return "rbac_sa_token_request" }

var edgePropertiesRBACSATokenRequest = map[string]any{
	"Description": "ServiceAccount has RBAC permission to create ServiceAccount tokens (TokenRequest), allowing it to mint API tokens for any ServiceAccount in scope.",
	"Reference":   "https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#bound-service-account-tokens",
}

func (r rbacSATokenRequestEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, saTokenRequestNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, saTokenRequestCluster(ctx)...)
	return edges
}

func saTokenRequestNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"serviceaccounts/token"}
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
			sa := resolveNamespacedSubjectSA(ctx, namespace, binding.Namespace, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			if all {
				if len(space.AllServiceAccounts) > 0 {
					agg := &space.AllServiceAccounts[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "SATokenRequest", edgePropertiesRBACSATokenRequest))
				}
				continue
			}
			for i := range space.ServiceAccounts {
				target := &space.ServiceAccounts[i]
				if _, ok := names[target.Name]; ok {
					edges = append(edges, framework.CreateEdgeWithProperties(sa, target, "SATokenRequest", edgePropertiesRBACSATokenRequest))
				}
			}
		}
	}
	return edges
}

func saTokenRequestCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"serviceaccounts/token"}
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
			sa := resolveClusterSubjectSA(ctx, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			if all {
				if len(ctx.Core.Cluster.AllServiceAccounts) > 0 {
					agg := &ctx.Core.Cluster.AllServiceAccounts[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "SATokenRequest", edgePropertiesRBACSATokenRequest))
				}
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for i := range space.ServiceAccounts {
					target := &space.ServiceAccounts[i]
					if _, ok := names[target.Name]; ok {
						edges = append(edges, framework.CreateEdgeWithProperties(sa, target, "SATokenRequest", edgePropertiesRBACSATokenRequest))
					}
				}
			}
		}
	}
	return edges
}
