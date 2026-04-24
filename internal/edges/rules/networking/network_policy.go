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
			for j := range space.Pods {
				pod := &space.Pods[j]
				if framework.LabelsMatchOnly(pod.LabelsMap, netpol.PodSelectorLabels) {
					edges = append(edges, framework.CreateEdge(netpol, pod, "AppliesTo"))
				}
			}
		}
	}
	return edges
}
