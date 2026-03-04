package edges

import "bloodhound-kube/internal/model"

type rbacCreateEdgesRule struct{}

// func init() {
// 	RegisterEdgeRule(rbacCreateEdgesRule{})
// }

func (r rbacCreateEdgesRule) Name() string {
	return "rbac_create"
}

var edgePropertiesRBACCreate = map[string]any{
	"Description": "ServiceAccount has RBAC permissions to create RoleBindings or ClusterRoleBindings",
	"Reference":   "",
}

func (r rbacCreateEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
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

// SA w/ create on RoleBindings/ClusterRoleBindings -> Role/ClusterRole that can be bound to other SAs.
func rbacCreateNamespaced(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"rolebindings", "rbac.authorization.k8s.io/rolebindings"}
	verbs := []string{"create"}

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

			for i := range space.Roles {
				role := &space.Roles[i]
				edges = append(edges, CreateEdgeWithProperties(sa, role, "RBACCreate", edgePropertiesRBACCreate))
			}
			for _, clusterRole := range ctx.Core.Cluster.ClusterRoles {
				cr := clusterRole
				edges = append(edges, CreateEdgeWithProperties(sa, &cr, "RBACCreate", edgePropertiesRBACCreate))
			}
		}
	}
	return edges
}

func rbacCreateCluster(ctx *EdgeContext) []model.BloodHoundEdge {
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
			for _, clusterRole := range ctx.Core.Cluster.ClusterRoles {
				cr := clusterRole
				edges = append(edges, CreateEdgeWithProperties(sa, &cr, "RBACCreate", edgePropertiesRBACCreate))
			}
		}
	}
	return edges
}
