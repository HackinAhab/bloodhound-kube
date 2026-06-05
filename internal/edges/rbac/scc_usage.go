package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type rbacSCCUsageEdgesRule struct{}

func (r rbacSCCUsageEdgesRule) Name() string { return "rbac_scc_usage" }

var edgePropertiesRBACSCCUsage = map[string]any{
	"Description": "ServiceAccount has RBAC permission to use this SecurityContextConstraints.",
	"Reference":   "https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html",
}

// Non-core resources appear in PermsDisplay as "<group>/<resource>: verbs",
// so we check both the bare name and the fully-qualified form.
var sccResourceKeys = []string{
	"securitycontextconstraints",
	"security.openshift.io/securitycontextconstraints",
}

func (r rbacSCCUsageEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	return sccUsageCluster(ctx)
}

func sccUsageCluster(ctx *framework.Context) []model.BloodHoundEdge {
	verbs := []string{"use"}
	var edges []model.BloodHoundEdge

	for _, binding := range ctx.Index.ClusterRoleBindingsByName {
		if binding.RoleKind != "ClusterRole" {
			continue
		}
		clusterRole := ctx.Index.ClusterRolesByName[binding.RoleName]
		if clusterRole == nil {
			continue
		}
		all, names := accessForResource(clusterRole.PermsDisplay, sccResourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			sa := resolveClusterSubjectSA(ctx, subject.Kind, subject.Namespace, subject.Name)
			if sa == nil {
				continue
			}
			if all {
				for i := range ctx.Core.Cluster.SecurityContextConstraints {
					scc := &ctx.Core.Cluster.SecurityContextConstraints[i]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, scc, "SCCUse", edgePropertiesRBACSCCUsage))
				}
				continue
			}
			for name := range names {
				scc := ctx.Index.SecurityContextConstraintsBy[name]
				if scc == nil {
					continue
				}
				edges = append(edges, framework.CreateEdgeWithProperties(sa, scc, "SCCUse", edgePropertiesRBACSCCUsage))
			}
		}
	}
	return edges
}
