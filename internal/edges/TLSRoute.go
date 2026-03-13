package edges

import "bloodhound-kube/internal/model"

type tlsRouteEdgesRule struct{}

func (r tlsRouteEdgesRule) Name() string {
	return "tlsroutes"
}

func init() {
	RegisterEdgeRule(tlsRouteEdgesRule{})
}

func (r tlsRouteEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.TLSRoutes {
			route := &space.TLSRoutes[i]
			for _, backend := range route.BackendRefs {
				backendNS := backend.Namespace
				backendName := backend.Name
				if backendName == "" {
					continue
				}
				if backendNS == "" {
					backendNS = ns
				}
				if serviceIndex := ctx.Index.ServicesByNamespace[backendNS]; serviceIndex != nil {
					if svc := serviceIndex[backendName]; svc != nil {
						edges = append(edges, CreateEdge(route, svc, "RoutesTo"))
					}
				}
			}
			if ctx.Index.External != nil {
				edges = append(edges, CreateEdge(ctx.Index.External, route, "ExternalRoutesTo"))
			}
		}
	}
	return edges
}
