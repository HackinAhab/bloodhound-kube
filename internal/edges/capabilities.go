package edges

import "bloodhound-kube/internal/model"

type capabilityEdgesRule struct{}

func (r capabilityEdgesRule) Name() string {
	return "capabilities"
}

func (r capabilityEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Pods {
			pod := &space.Pods[i]
			if pod.NodeName == "" || pod.ID == "" {
				continue
			}
			node := ctx.Index.NodesByName[pod.NodeName]
			if node == nil || node.ID == "" {
				continue
			}
			for _, capAdd := range pod.CapabilitiesAdd {
				norm := NormalizeCapability(capAdd)
				info, ok := capabilityDescriptions[norm]
				if !ok {
					continue
				}
				props := map[string]any{
					"Description": info.Description,
					"Reference":   info.Reference,
				}
				edges = append(edges, CreateEdgeWithProperties(pod, node, norm, props))
			}
		}
	}
	return edges
}

func init() {
	RegisterEdgeRule(capabilityEdgesRule{})
}
