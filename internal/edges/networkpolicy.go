package edges

import "bloodhound-kube/internal/model"

type networkPolicyEdgesRule struct{}

func (r networkPolicyEdgesRule) Name() string {
	return "networkpolicy"
}

func (r networkPolicyEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
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
				if IsSubset(netpol.PodSelector, pod.LabelsMap) {
					edges = append(edges, CreateEdge(netpol, pod, "AppliesTo"))
				}
			}
		}
	}
	return edges
}

func init() {
	RegisterEdgeRule(networkPolicyEdgesRule{})
}
