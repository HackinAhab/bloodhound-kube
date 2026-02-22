package edges

import "bloodhound-kube/internal/model"

type ingressEdgesRule struct{}

func (r ingressEdgesRule) Name() string {
	return "ingress"
}

func (r ingressEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		serviceIndex := ctx.Index.ServicesByNamespace[ns]
		for i := range space.Ingresses {
			ingress := &space.Ingresses[i]
			for _, backend := range ingress.BackendServices {
				if serviceIndex == nil {
					continue
				}
				if svc := serviceIndex[backend]; svc != nil {
					edges = append(edges, CreateEdge(ingress, svc, "RoutesTo"))
				}
			}
			if ctx.Index.External != nil {
				edges = append(edges, CreateEdge(ctx.Index.External, ingress, "ExternalRoutesTo"))
			}
		}
	}
	return edges
}

func init() {
	RegisterEdgeRule(ingressEdgesRule{})
}
