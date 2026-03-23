package edges

import "bloodhound-kube/internal/model"

type daemonSetEdgesRule struct{}

func (r daemonSetEdgesRule) Name() string {
	return "daemonset"
}

func init() {
	RegisterEdgeRule(daemonSetEdgesRule{})
}

func (r daemonSetEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.DaemonSets {
			daemonSet := &space.DaemonSets[i]
			for j := range space.Pods {
				pod := &space.Pods[j]
				if labelsMatchOnly(pod.LabelsMap, daemonSet.SelectorLabels) {
					edges = append(edges, CreateEdge(daemonSet, pod, "ManagedBy"))
				}
			}
		}
	}
	return edges
}
