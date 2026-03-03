package edges

import "bloodhound-kube/internal/model"

type rbacPodDebugEdgesRule struct{}

func init() {
	RegisterEdgeRule(rbacPodDebugEdgesRule{})
}

func (r rbacPodDebugEdgesRule) Name() string {
	return "rbac_pod_debug"
}

func (r rbacPodDebugEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, podDebugNamespaced(ctx, ns, space)...)
	}
	edges = append(edges, podDebugCluster(ctx)...)
	return edges
}

// SA w/ debug -> Pods that can be debugged by the SA
func podDebugNamespaced(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	resourceKeys := []string{"pods/debug"}
	verbs := []string{"get"}

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
			for i := range space.Pods {
				pod := &space.Pods[i]
				if all {
					edges = append(edges, CreateEdge(sa, pod, "PodDebug"))
					continue
				}
				if names != nil {
					if _, ok := names[pod.Name]; ok {
						edges = append(edges, CreateEdge(sa, pod, "PodDebug"))
					}
				}
			}
		}
	}
	return edges
}

func podDebugCluster(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"pods/debug"}
	verbs := []string{"get"}

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
				for i := range space.Pods {
					pod := &space.Pods[i]
					if all {
						edges = append(edges, CreateEdge(sa, pod, "PodDebug"))
						continue
					}
					if names != nil {
						if _, ok := names[pod.Name]; ok {
							edges = append(edges, CreateEdge(sa, pod, "PodDebug"))
						}
					}
				}
			}
		}
	}
	return edges
}
