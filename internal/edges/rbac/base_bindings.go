package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type rbacEdgesRule struct{}

func (r rbacEdgesRule) Name() string { return "rbac_base" }

func (r rbacEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, roleToServiceAccountFromRoleBinding(ctx, ns)...)
		edges = append(edges, clusterRoleToServiceAccountFromRoleBinding(ctx, ns)...)
	}
	edges = append(edges, clusterRoleToServiceAccountFromClusterRoleBinding(ctx)...)
	return edges
}

func clusterRoleToServiceAccountFromRoleBinding(ctx *framework.Context, namespace string) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	for _, binding := range roleBindings {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		for _, subject := range binding.Subjects {
			if subject.Kind != "ServiceAccount" {
				continue
			}
			subjectNS := subject.Namespace
			if subjectNS == "" {
				subjectNS = binding.Namespace
			}
			if saIndex := ctx.Index.ServiceAccountsByNamespace[subjectNS]; saIndex != nil {
				if sa := saIndex[subject.Name]; sa != nil {
					edges = append(edges, framework.CreateEdge(clusterRole, sa, "RoleBound"))
				}
			}
		}
	}
	return edges
}

func clusterRoleToServiceAccountFromClusterRoleBinding(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		for _, subject := range binding.Subjects {
			if subject.Kind != "ServiceAccount" || subject.Namespace == "" {
				continue
			}
			if saIndex := ctx.Index.ServiceAccountsByNamespace[subject.Namespace]; saIndex != nil {
				if sa := saIndex[subject.Name]; sa != nil {
					edges = append(edges, framework.CreateEdge(clusterRole, sa, "RoleBound"))
				}
			}
		}
	}
	return edges
}

func roleToServiceAccountFromRoleBinding(ctx *framework.Context, namespace string) []model.BloodHoundEdge {
	if ctx == nil {
		return nil
	}
	serviceAccounts := ctx.Index.ServiceAccountsByNamespace[namespace]
	roleIndex := ctx.Index.RolesByNamespace[namespace]
	bindingIndex := ctx.Index.RoleBindingsByNamespace[namespace]
	if len(bindingIndex) == 0 || len(roleIndex) == 0 || len(serviceAccounts) == 0 {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, binding := range bindingIndex {
		if binding.RoleKind != "Role" {
			continue
		}
		role := roleIndex[binding.RoleName]
		if role == nil {
			continue
		}
		for _, subject := range binding.Subjects {
			if subject.Kind != "ServiceAccount" {
				continue
			}
			subjectNS := subject.Namespace
			if subjectNS == "" {
				subjectNS = binding.Namespace
			}
			if subjectNS != namespace {
				continue
			}
			if sa := serviceAccounts[subject.Name]; sa != nil {
				edges = append(edges, framework.CreateEdge(role, sa, "RoleBound"))
			}
		}
	}
	return edges
}
