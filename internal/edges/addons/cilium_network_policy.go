package addons

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type ciliumNetworkPolicyEdgesRule struct{}

func (r ciliumNetworkPolicyEdgesRule) Name() string { return "ciliumnetworkpolicy" }

func (r ciliumNetworkPolicyEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.CiliumNetworkPolicies {
			policy := &space.CiliumNetworkPolicies[i]
			if len(policy.PodSelectorLabels) == 0 {
				if agg := framework.FirstEdgeNode(space.AllPods); agg != nil {
					edges = append(edges, framework.CreateEdge(policy, agg, "BHK_AppliesTo"))
				}
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
