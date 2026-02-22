package edges

import "bloodhound-kube/internal/model"

type rbacEdgesRule struct{}

func (r rbacEdgesRule) Name() string {
	return "rbac"
}

func (r rbacEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge

	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}

		serviceAccounts := ctx.Index.ServiceAccountsByNamespace[ns]
		roleIndex := ctx.Index.RolesByNamespace[ns]
		bindingIndex := ctx.Index.RoleBindingsByNamespace[ns]

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
				if subjectNS != ns {
					continue
				}
				if sa := serviceAccounts[subject.Name]; sa != nil {
					edges = append(edges, CreateEdgeVia(role, sa, binding, "PermissionsFromRole"))
				}
			}
		}

		for i := range space.Secrets {
			secret := &space.Secrets[i]
			if secret.SecretType != "kubernetes.io/service-account-token" {
				continue
			}
			saName := ""
			if secret.AnnotationsMap != nil {
				if name, ok := secret.AnnotationsMap["kubernetes.io/service-account.name"].(string); ok {
					saName = name
				}
			}
			if saName == "" {
				continue
			}
			if sa := serviceAccounts[saName]; sa != nil {
				edges = append(edges, CreateEdge(sa, secret, "SAToken"))
			}
		}
	}

	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
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
			if subject.Namespace == "" {
				continue
			}
			if saIndex := ctx.Index.ServiceAccountsByNamespace[subject.Namespace]; saIndex != nil {
				if sa := saIndex[subject.Name]; sa != nil {
					edges = append(edges, CreateEdgeVia(clusterRole, sa, binding, "PermissionsFromRole"))
				}
			}
		}
	}

	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		roleBindings := ctx.Index.RoleBindingsByNamespace[ns]
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
						edges = append(edges, CreateEdgeVia(clusterRole, sa, binding, "PermissionsFromRole"))
					}
				}
			}
		}
	}

	return edges
}

func init() {
	RegisterEdgeRule(rbacEdgesRule{})
}
