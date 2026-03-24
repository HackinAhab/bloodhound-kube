package edges

import "bloodhound-kube/internal/model"

type tcpRouteEdgesRule struct{}

func (r tcpRouteEdgesRule) Name() string {
	return "tcproutes"
}

func init() {
	RegisterEdgeRule(tcpRouteEdgesRule{})
}

func (r tcpRouteEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.TCPRoutes {
			route := &space.TCPRoutes[i]
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
