//go:build !no_addons && !no_calico

package calico

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

func Register(reg *framework.Registry) {
	reg.Register(globalNetworkPolicyEdgesRule{})
}

type globalNetworkPolicyEdgesRule struct{}

func (r globalNetworkPolicyEdgesRule) Name() string { return "globalnetworkpolicy" }

func (r globalNetworkPolicyEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil || ctx.Core.Cluster == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for i := range ctx.Core.Cluster.GlobalNetworkPolicies {
		policy := &ctx.Core.Cluster.GlobalNetworkPolicies[i]
		if !policy.SelectorRecognized {
			continue
		}
		if policy.MatchesAll {
			if agg := framework.FirstEdgeNode(ctx.Core.Cluster.AllPods); agg != nil {
				edges = append(edges, framework.CreateEdge(policy, agg, "BHK_AppliesTo"))
			}
		} else {
			for _, space := range ctx.Core.Namespaces {
				if space == nil {
					continue
				}
				for j := range space.Pods {
					pod := &space.Pods[j]
					if framework.LabelsMatchOnly(pod.LabelsMap, policy.SelectorLabels) {
						edges = append(edges, framework.CreateEdge(policy, pod, "BHK_AppliesTo"))
					}
				}
			}
		}

		// GlobalNetworkPolicy selectors also match Calico HostEndpoints (host
		// network interfaces), which are resolved to the underlying BHK_Node via
		// spec.node. HostEndpoint has no graph node of its own; an empty selector
		// still only reaches nodes that actually have a HostEndpoint object, so
		// there's no "matches all nodes" aggregate shortcut here.
		for _, he := range ctx.Core.Cluster.HostEndpoints {
			if !framework.LabelsMatchOnly(he.LabelsMap, policy.SelectorLabels) {
				continue
			}
			if node := ctx.Index.NodesByName[he.NodeName]; node != nil {
				edges = append(edges, framework.CreateEdge(policy, node, "BHK_AppliesTo"))
			}
		}
	}
	return edges
}
