package networking

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type networkPolicyEdgesRule struct{}

func (r networkPolicyEdgesRule) Name() string { return "networkpolicy" }

func (r networkPolicyEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.NetworkPolicies {
			netpol := &space.NetworkPolicies[i]
			if len(netpol.PodSelectorLabels) == 0 {
				if agg := framework.FirstEdgeNode(space.AllPods); agg != nil {
					edges = append(edges, framework.CreateEdge(netpol, agg, "BHK_AppliesTo"))
				}
				continue
			}
			for j := range space.Pods {
				pod := &space.Pods[j]
				if framework.LabelsMatchOnly(pod.LabelsMap, netpol.PodSelectorLabels) {
					edges = append(edges, framework.CreateEdge(netpol, pod, "BHK_AppliesTo"))
				}
			}
		}
	}
	return edges
}
