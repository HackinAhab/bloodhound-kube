package networking

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type routeEdgesRule struct{}

func (r routeEdgesRule) Name() string { return "route" }

func (r routeEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Routes {
			rt := &space.Routes[i]
			edges = append(edges, routeBackendsToServices(ctx, rt, ns, rt.BackendRefs)...)
			if ctx.Index.External != nil {
				edges = append(edges, framework.CreateEdge(ctx.Index.External, rt, "BHK_ExternalRoutesTo"))
			}
		}
	}
	return edges
}
