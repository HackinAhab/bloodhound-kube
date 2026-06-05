package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

// rbacEscalateBindEdgesRule covers the `escalate` and `bind` RBAC verbs on
// roles/clusterroles. Both are privilege-escalation primitives:
//   - escalate: update a role to include permissions the caller doesn't hold
//   - bind:     bind a role to any subject without holding the role's permissions
type rbacEscalateBindEdgesRule struct{}

func (r rbacEscalateBindEdgesRule) Name() string { return "rbac_escalate_bind" }

var edgePropertiesRBACEscalate = map[string]any{
	"Description": "ServiceAccount has the 'escalate' verb on roles/clusterroles, allowing it to grant itself permissions it does not currently hold by modifying the role definition.",
	"Reference":   "https://kubernetes.io/docs/reference/access-authn-authz/rbac/#privilege-escalation-prevention-and-bootstrapping",
}

var edgePropertiesRBACBind = map[string]any{
	"Description": "ServiceAccount has the 'bind' verb on roles/clusterroles, allowing it to create role bindings for roles it does not hold.",
	"Reference":   "https://kubernetes.io/docs/reference/access-authn-authz/rbac/#privilege-escalation-prevention-and-bootstrapping",
}

func (r rbacEscalateBindEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, escalateBindNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, escalateBindCluster(ctx)...)
	return edges
}

func escalateBindNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	roleResourceKeys := []string{"roles", "rbac.authorization.k8s.io/roles"}
	clusterRoleResourceKeys := []string{"clusterroles", "rbac.authorization.k8s.io/clusterroles"}

	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	var edges []model.BloodHoundEdge
	for _, binding := range roleBindings {
		perms := permsForBinding(ctx, namespace, binding.RoleKind, binding.RoleName)
		if len(perms) == 0 {
			continue
		}
		parsed := parseRBACPerms(perms)
		canEscalateRole, _ := accessForParsedResource(parsed, roleResourceKeys, []string{"escalate"})
		canBindRole, _ := accessForParsedResource(parsed, roleResourceKeys, []string{"bind"})
		canEscalateCR, _ := accessForParsedResource(parsed, clusterRoleResourceKeys, []string{"escalate"})
		canBindCR, _ := accessForParsedResource(parsed, clusterRoleResourceKeys, []string{"bind"})
		if !canEscalateRole && !canBindRole && !canEscalateCR && !canBindCR {
			continue
		}
		for _, subject := range binding.Subjects {
			sa := resolveNamespacedSubjectSA(ctx, namespace, binding.Namespace, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			if canEscalateRole || canBindRole {
				edgeType := rbacEscalateBindEdgeType(canEscalateRole)
				props := rbacEscalateBindProps(canEscalateRole)
				for i := range space.Roles {
					role := &space.Roles[i]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, role, edgeType, props))
				}
			}
			if canEscalateCR || canBindCR {
				edgeType := rbacEscalateBindEdgeType(canEscalateCR)
				props := rbacEscalateBindProps(canEscalateCR)
				for _, clusterRole := range ctx.Core.Cluster.ClusterRoles {
					cr := clusterRole
					edges = append(edges, framework.CreateEdgeWithProperties(sa, &cr, edgeType, props))
				}
			}
		}
	}
	return edges
}

func escalateBindCluster(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	clusterRoleResourceKeys := []string{"clusterroles", "rbac.authorization.k8s.io/clusterroles"}
	roleResourceKeys := []string{"roles", "rbac.authorization.k8s.io/roles"}

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
		canEscalateCR, _ := accessForParsedResource(parsed, clusterRoleResourceKeys, []string{"escalate"})
		canBindCR, _ := accessForParsedResource(parsed, clusterRoleResourceKeys, []string{"bind"})
		canEscalateRole, _ := accessForParsedResource(parsed, roleResourceKeys, []string{"escalate"})
		canBindRole, _ := accessForParsedResource(parsed, roleResourceKeys, []string{"bind"})
		if !canEscalateCR && !canBindCR && !canEscalateRole && !canBindRole {
			continue
		}
		for _, subject := range binding.Subjects {
			sa := resolveClusterSubjectSA(ctx, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			if canEscalateCR || canBindCR {
				edgeType := rbacEscalateBindEdgeType(canEscalateCR)
				props := rbacEscalateBindProps(canEscalateCR)
				for _, cr := range ctx.Core.Cluster.ClusterRoles {
					c := cr
					edges = append(edges, framework.CreateEdgeWithProperties(sa, &c, edgeType, props))
				}
			}
			if canEscalateRole || canBindRole {
				edgeType := rbacEscalateBindEdgeType(canEscalateRole)
				props := rbacEscalateBindProps(canEscalateRole)
				for _, space := range ctx.Core.Namespaces {
					if space == nil {
						continue
					}
					for i := range space.Roles {
						role := &space.Roles[i]
						edges = append(edges, framework.CreateEdgeWithProperties(sa, role, edgeType, props))
					}
				}
			}
		}
	}
	return edges
}

// rbacEscalateBindEdgeType returns "RBACEscalate" when escalate is present,
// "RBACBind" when only bind is present. Escalate is the more severe verb.
func rbacEscalateBindEdgeType(canEscalate bool) string {
	if canEscalate {
		return "RBACEscalate"
	}
	return "RBACBind"
}

func rbacEscalateBindProps(isEscalate bool) map[string]any {
	if isEscalate {
		return edgePropertiesRBACEscalate
	}
	return edgePropertiesRBACBind
}
