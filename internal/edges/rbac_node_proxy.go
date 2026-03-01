package edges

import "bloodhound-kube/internal/model"

type rbacNodeProxyEdgesRule struct{}

// func init() {
// 	RegisterEdgeRule(rbacNodeProxyEdgesRule{})
// }

func (r rbacNodeProxyEdgesRule) Name() string {
	return "rbac_node_proxy"
}

func (r rbacNodeProxyEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, rbacNodeProxyToPodNamespaced(ctx, ns)...)
	}
	edges = append(edges, rbacNodeProxyToPodCluster(ctx)...)
	return edges
}

// SA w/ node/proxy -> All pods on the same node (if only scopped for a single node, otherwise all pods in the cluster)
// Based on https://grahamhelton.com/blog/nodes-proxy-rce
func rbacNodeProxyToPodNamespaced(ctx *EdgeContext, namespace string) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"nodes/proxy"}
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

			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for i := range space.Pods {
					pod := &space.Pods[i]
					if pod.NodeName == "" {
						continue
					}
					if all {
						edges = append(edges, CreateEdge(sa, pod, "NodeProxy"))
						continue
					}
					if names != nil {
						if _, ok := names[pod.NodeName]; ok {
							edges = append(edges, CreateEdge(sa, pod, "NodeProxy"))
						}
					}
				}
			}
		}
	}
	return edges
}

func rbacNodeProxyToPodCluster(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"nodes/proxy"}
	verbs := []string{"get", "create", "proxy"}

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
					if pod.NodeName == "" {
						continue
					}
					if all {
						edges = append(edges, CreateEdge(sa, pod, "NodeProxyRCE"))
						continue
					}
					if names != nil {
						if _, ok := names[pod.NodeName]; ok {
							edges = append(edges, CreateEdge(sa, pod, "NodeProxyRCE"))
						}
					}
				}
			}
		}
	}
	return edges
}
