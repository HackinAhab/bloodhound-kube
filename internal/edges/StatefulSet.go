package edges

import "bloodhound-kube/internal/model"

type statefulSetEdgesRule struct{}

func (r statefulSetEdgesRule) Name() string {
	return "statefulset"
}

func init() {
	RegisterEdgeRule(statefulSetEdgesRule{})
}

func (r statefulSetEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.StatefulSets {
			statefulSet := &space.StatefulSets[i]
			for j := range space.Pods {
				pod := &space.Pods[j]
				if labelsMatchOnly(pod.LabelsMap, statefulSet.SelectorLabels) {
					edges = append(edges, CreateEdge(statefulSet, pod, "ManagedBy"))
				}
			}
		}
	}
	return edges
}
