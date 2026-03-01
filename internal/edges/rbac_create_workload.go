package edges

import "bloodhound-kube/internal/model"

type rbacCreateWorkloadEdgesRule struct{}

// func init() {
// 	RegisterEdgeRule(rbacCreateWorkloadEdgesRule{})
// }

func (r rbacCreateWorkloadEdgesRule) Name() string {
	return "rbac_create_workload"
}

// SA w/ create on workloads -> Nodes that can be scheduled on
func workloadCreateNamespaced(ctx *EdgeContext, namespace string) []model.BloodHoundEdge {
	if ctx == nil {
		return nil
	}
	resourceKeys := []string{
		"pods",
		"deployments",
		"apps/deployments",
		"daemonsets",
		"apps/daemonsets",
		"statefulsets",
		"apps/statefulsets",
	}
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

			for i := range ctx.Core.Cluster.Nodes {
				node := &ctx.Core.Cluster.Nodes[i]
				edges = append(edges, CreateEdge(sa, node, "WorkloadCreate"))
			}
		}
	}
	return edges
}

func workloadCreateCluster(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{
		"pods",
		"deployments",
		"apps/deployments",
		"daemonsets",
		"apps/daemonsets",
		"statefulsets",
		"apps/statefulsets",
	}
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
			for i := range ctx.Core.Cluster.Nodes {
				node := &ctx.Core.Cluster.Nodes[i]
				edges = append(edges, CreateEdge(sa, node, "WorkloadCreate"))
			}
		}
	}
	return edges
}
