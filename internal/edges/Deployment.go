package edges

import "bloodhound-kube/internal/model"

type deploymentEdgesRule struct{}

func (r deploymentEdgesRule) Name() string {
	return "deployment"
}

func (r deploymentEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Deployments {
			deploy := &space.Deployments[i]
			for j := range space.Pods {
				pod := &space.Pods[j]
				if labelsMatchOnly(pod.LabelsMap, deploy.SelectorLabels) {
					edges = append(edges, CreateEdge(deploy, pod, "ManagedBy"))
				}
			}
		}
	}
	return edges
}

func init() {
	RegisterEdgeRule(deploymentEdgesRule{})
}
