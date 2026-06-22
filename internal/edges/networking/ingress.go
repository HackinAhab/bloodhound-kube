package networking

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type ingressEdgesRule struct{}

func (r ingressEdgesRule) Name() string { return "ingress" }

func (r ingressEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Ingresses {
			ingress := &space.Ingresses[i]
			edges = append(edges, routeBackendsToServices(ctx, ingress, ns, ingress.BackendRefs)...)
			if ctx.Index.External != nil {
				edges = append(edges, framework.CreateEdge(ctx.Index.External, ingress, "BHK_ExternalRoutesTo"))
			}
		}
	}
	return edges
}
