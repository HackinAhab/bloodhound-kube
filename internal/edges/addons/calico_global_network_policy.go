package addons

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

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
			continue
		}
		for _, space := range ctx.Core.Namespaces {
			if space == nil {
				continue
			}
			for j := range space.Pods {
				pod := &space.Pods[j]
				if framework.LabelsMatchOnly(pod.LabelsMap, policy.PodSelectorLabels) {
					edges = append(edges, framework.CreateEdge(policy, pod, "BHK_AppliesTo"))
				}
			}
		}
	}
	return edges
}
