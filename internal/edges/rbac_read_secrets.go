package edges

import "bloodhound-kube/internal/model"

type rbacReadSecretsEdgesRule struct{}

func init() {
	RegisterEdgeRule(rbacReadSecretsEdgesRule{})
}

func (r rbacReadSecretsEdgesRule) Name() string {
	return "rbac_read_secrets"
}

func (r rbacReadSecretsEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, saReadSecretNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, saReadSecretCluster(ctx)...)
	return edges
}

// SA w/ read access to secrets -> Secrets
func saReadSecretNamespaced(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"secrets"}
	verbs := []string{"get", "list", "watch"}

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
			for i := range space.Secrets {
				secret := &space.Secrets[i]
				if all {
					edges = append(edges, CreateEdge(sa, secret, "SAReadSecret"))
					continue
				}
				if names != nil {
					if _, ok := names[secret.Name]; ok {
						edges = append(edges, CreateEdge(sa, secret, "SAReadSecret"))
					}
				}
			}
		}
	}
	return edges
}

func saReadSecretCluster(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"secrets"}
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
			if all {
				if len(ctx.Core.Cluster.AllSecrets) > 0 {
					agg := &ctx.Core.Cluster.AllSecrets[0]
					edges = append(edges, CreateEdge(sa, agg, "SAReadSecret"))
				}
				continue
			}
			if len(names) > 0 {
				for _, space := range ctx.Core.Namespaces {
					if space == nil {
						continue
					}
					for i := range space.Secrets {
						secret := &space.Secrets[i]
						if _, ok := names[secret.Name]; ok {
							edges = append(edges, CreateEdge(sa, secret, "SAReadSecret"))
						}
					}
				}
			}
		}
	}
	return edges
}
