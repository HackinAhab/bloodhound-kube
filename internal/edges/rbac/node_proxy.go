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
	return rbacNodeProxyToPodCluster(ctx)
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
			principal := resolveClusterSubject(ctx, subject)
			if principal == nil {
				continue
			}
			if all {
				if len(ctx.Core.Cluster.AllPods) > 0 {
					agg := &ctx.Core.Cluster.AllPods[0]
					edges = append(edges, framework.CreateEdge(principal, agg, "BHK_NodeProxyRCE"))
				}
			} else {
				for _, space := range ctx.Core.Namespaces {
					if space == nil {
						continue
					}
					for i := range space.Pods {
						pod := &space.Pods[i]
						if pod.NodeName == "" {
							continue
						}
						if hasName(names, pod.NodeName) {
							edges = append(edges, framework.CreateEdge(principal, pod, "BHK_NodeProxyRCE"))
						}
					}
				}
			}
			// Also emit SA → Node edges so the node compromise path is visible
			// directly without traversing through pod intermediaries.
			if all {
				if len(ctx.Core.Cluster.AllNodes) > 0 {
					agg := &ctx.Core.Cluster.AllNodes[0]
					edges = append(edges, framework.CreateEdgeWithProperties(principal, agg, "BHK_NodeProxyRCE", edgePropertiesRBACNodeProxy))
				}
			} else {
				for i := range ctx.Core.Cluster.Nodes {
					node := &ctx.Core.Cluster.Nodes[i]
					if hasName(names, node.Name) {
						edges = append(edges, framework.CreateEdgeWithProperties(principal, node, "BHK_NodeProxyRCE", edgePropertiesRBACNodeProxy))
					}
				}
			}
		}
	}
	return edges
}
