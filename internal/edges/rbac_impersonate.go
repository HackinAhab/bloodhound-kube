package edges

import "bloodhound-kube/internal/model"

type rbacImpersonateEdgesRule struct{}

func init() {
	RegisterEdgeRule(rbacImpersonateEdgesRule{})
}

func (r rbacImpersonateEdgesRule) Name() string {
	return "rbac_impersonate"
}

var edgePropertiesRBACImpersonate = map[string]any{
	"Description": "ServiceAccount has RBAC permissions to impersonate another ServiceAccount",
	"Reference":   "https://kubehound.io/reference/attacks/IDENTITY_IMPERSONATE/",
}

func (r rbacImpersonateEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
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
					edges = append(edges, CreateEdgeWithProperties(sa, target, "SAImpersonate", edgePropertiesRBACImpersonate))
					continue
				}
				if names != nil {
					if _, ok := names[target.Name]; ok {
						edges = append(edges, CreateEdgeWithProperties(sa, target, "SAImpersonate", edgePropertiesRBACImpersonate))
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
			if all {
				if len(ctx.Core.Cluster.AllServiceAccounts) > 0 {
					agg := &ctx.Core.Cluster.AllServiceAccounts[0]
					edges = append(edges, CreateEdgeWithProperties(sa, agg, "SAImpersonate", edgePropertiesRBACImpersonate))
				}
				continue
			}
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for i := range space.ServiceAccounts {
					target := &space.ServiceAccounts[i]
					if names != nil {
						if _, ok := names[target.Name]; ok {
							edges = append(edges, CreateEdgeWithProperties(sa, target, "SAImpersonate", edgePropertiesRBACImpersonate))
						}
					}
				}
			}
		}
	}
	return edges
}
