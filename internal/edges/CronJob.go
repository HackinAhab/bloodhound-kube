package edges

import "bloodhound-kube/internal/model"

type cronJobEdgesRule struct{}

func (r cronJobEdgesRule) Name() string {
	return "cronjob"
}

func init() {
	RegisterEdgeRule(cronJobEdgesRule{})
}

func (r cronJobEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.CronJobs {
			cronJob := &space.CronJobs[i]
			if len(cronJob.SelectorLabels) == 0 {
				continue
			}
			for j := range space.Pods {
				pod := &space.Pods[j]
				if labelsMatchOnly(pod.LabelsMap, cronJob.SelectorLabels) {
					edges = append(edges, CreateEdge(cronJob, pod, "ManagedBy"))
				}
			}
		}
	}
	return edges
}
