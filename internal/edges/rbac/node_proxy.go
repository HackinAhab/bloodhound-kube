package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type rbacNodeProxyEdgesRule struct{}

func (r rbacNodeProxyEdgesRule) Name() string { return "rbac_node_proxy" }

var edgePropertiesRBACNodeProxy = map[string]any{
	"Description": "ServiceAccount has RBAC permissions to proxy to nodes, which can allow for lateral movement and potential RCE if the permissions include 'get' verb",
	"Reference":   "https://grahamhelton.com/blog/nodes-proxy-rce",
}

func (r rbacNodeProxyEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
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

func rbacNodeProxyToPodNamespaced(ctx *framework.Context, namespace string) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	resourceKeys := []string{"nodes/proxy"}
	verbs := []string{"get"}

	roleBindings := ctx.Index.RoleBindingsByNamespace[namespace]
	var edges []model.BloodHoundEdge
	for _, binding := range roleBindings {
		perms := permsForBinding(ctx, namespace, binding.RoleKind, binding.RoleName)
		if len(perms) == 0 {
			continue
		}
		all, names := accessForResource(perms, resourceKeys, verbs)
		if !all && len(names) == 0 {
			continue
		}
		for _, subject := range binding.Subjects {
			sa := resolveNamespacedSubjectSA(ctx, namespace, binding.Namespace, subject.Kind, subject.Namespace, subject.Name)
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
						edges = append(edges, framework.CreateEdge(sa, pod, "NodeProxy"))
						continue
					}
					if _, ok := names[pod.NodeName]; ok {
						edges = append(edges, framework.CreateEdgeWithProperties(sa, pod, "NodeProxy", edgePropertiesRBACNodeProxy))
					}
				}
			}
		}
	}
	return edges
}

func rbacNodeProxyToPodCluster(ctx *framework.Context) []model.BloodHoundEdge {
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
			sa := resolveClusterSubjectSA(ctx, subject.Kind, subject.Namespace, subject.Name)
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
					if all || hasName(names, pod.NodeName) {
						edges = append(edges, framework.CreateEdge(sa, pod, "NodeProxyRCE"))
					}
				}
			}
			// Also emit SA → Node edges so the node compromise path is visible
			// directly without traversing through pod intermediaries.
			if all {
				if len(ctx.Core.Cluster.AllNodes) > 0 {
					agg := &ctx.Core.Cluster.AllNodes[0]
					edges = append(edges, framework.CreateEdgeWithProperties(sa, agg, "NodeProxyRCE", edgePropertiesRBACNodeProxy))
				}
			} else {
				for i := range ctx.Core.Cluster.Nodes {
					node := &ctx.Core.Cluster.Nodes[i]
					if hasName(names, node.Name) {
						edges = append(edges, framework.CreateEdgeWithProperties(sa, node, "NodeProxyRCE", edgePropertiesRBACNodeProxy))
					}
				}
			}
		}
	}
	return edges
}
