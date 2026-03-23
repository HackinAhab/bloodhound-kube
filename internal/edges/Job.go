package edges

import "bloodhound-kube/internal/model"

type jobEdgesRule struct{}

func (r jobEdgesRule) Name() string {
	return "job"
}

func init() {
	RegisterEdgeRule(jobEdgesRule{})
}

func (r jobEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Jobs {
			job := &space.Jobs[i]
			if len(job.SelectorLabels) == 0 {
				continue
			}
			for j := range space.Pods {
				pod := &space.Pods[j]
				if labelsMatchOnly(pod.LabelsMap, job.SelectorLabels) {
					edges = append(edges, CreateEdge(job, pod, "ManagedBy"))
				}
			}
		}
	}
	return edges
}
