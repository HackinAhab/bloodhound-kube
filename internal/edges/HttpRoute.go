package edges

import "bloodhound-kube/internal/model"

type httpRouteEdgesRule struct{}

func (r httpRouteEdgesRule) Name() string {
	return "httproutes"
}

func init() {
	RegisterEdgeRule(httpRouteEdgesRule{})
}

func (r httpRouteEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.HTTPRoutes {
			route := &space.HTTPRoutes[i]
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
		}
	}
	return edges
}
