package edges

import "bloodhound-kube/internal/model"

// SA w/ impersonate -> SA that can be impersonated
func saImpersonateNamespaced(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"serviceaccounts"}
	verbs := []string{"impersonate"}

	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	var edges []model.BloodHoundEdge
	for _, binding := range roleBindings {
		var perms []string
		switch binding.RoleKind {
		case "Role":
			roleIndex := ctx.Index.RolesByNamespace[namespace]
			if roleIndex == nil {
				continue
			}
			role := roleIndex[binding.RoleName]
			if role == nil {
				continue
			}
			perms = role.PermsDisplay
		case "ClusterRole":
			clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
			if clusterRole == nil {
				continue
			}
			perms = clusterRole.PermsDisplay
		default:
			continue
		}

		all, names := accessForResource(perms, resourceKeys, verbs)
		if !all && len(names) == 0 {
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
			saIndex := ctx.Index.ServiceAccountsByNamespace[subjectNS]
			if saIndex == nil {
				continue
			}
			sa := saIndex[subject.Name]
			if sa == nil {
				continue
			}

			for i := range space.ServiceAccounts {
				target := &space.ServiceAccounts[i]
				if all {
					edges = append(edges, CreateEdge(sa, target, "SAImpersonate"))
					continue
				}
				if names != nil {
					if _, ok := names[target.Name]; ok {
						edges = append(edges, CreateEdge(sa, target, "SAImpersonate"))
					}
				}
			}
		}
	}
	return edges
}

func saImpersonateCluster(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"serviceaccounts"}
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
		all, names := accessForResource(clusterRole.PermsDisplay, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			if subject.Kind != "ServiceAccount" || subject.Namespace == "" {
				continue
			}
			saIndex := ctx.Index.ServiceAccountsByNamespace[subject.Namespace]
			if saIndex == nil {
				continue
			}
			sa := saIndex[subject.Name]
			if sa == nil {
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for i := range space.ServiceAccounts {
					target := &space.ServiceAccounts[i]
					if all {
						edges = append(edges, CreateEdge(sa, target, "SAImpersonate"))
						continue
					}
					if names != nil {
						if _, ok := names[target.Name]; ok {
							edges = append(edges, CreateEdge(sa, target, "SAImpersonate"))
						}
					}
				}
			}
		}
	}
	return edges
}
