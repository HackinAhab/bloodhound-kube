package edges

import (
	"strings"

	"bloodhound-kube/internal/model"
)

type httpRouteEdgesRule struct{}

func (r httpRouteEdgesRule) Name() string {
	return "httproutes"
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
			for _, key := range route.BackendRefKeys {
				parts := strings.SplitN(key, "/", 2)
				backendNS := ""
				backendName := ""
				if len(parts) == 2 {
					backendNS = parts[0]
					backendName = parts[1]
				} else if len(parts) == 1 {
					backendName = parts[0]
				}
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

func init() {
	RegisterEdgeRule(httpRouteEdgesRule{})
}
