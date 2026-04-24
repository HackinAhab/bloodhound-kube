package networking

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

type httpRouteEdgesRule struct{}

func (r httpRouteEdgesRule) Name() string { return "httproutes" }

func (r httpRouteEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
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
			edges = append(edges, routeBackendsToServices(ctx, route, ns, route.BackendRefs)...)
		}
	}
	return edges
}

type grpcRouteEdgesRule struct{}

func (r grpcRouteEdgesRule) Name() string { return "grpcroutes" }

func (r grpcRouteEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.GRPCRoutes {
			route := &space.GRPCRoutes[i]
			edges = append(edges, routeBackendsToServices(ctx, route, ns, route.BackendRefs)...)
		}
	}
	return edges
}

type tcpRouteEdgesRule struct{}

func (r tcpRouteEdgesRule) Name() string { return "tcproutes" }

func (r tcpRouteEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
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
			edges = append(edges, routeBackendsToServices(ctx, route, ns, route.BackendRefs)...)
		}
	}
	return edges
}

type tlsRouteEdgesRule struct{}

func (r tlsRouteEdgesRule) Name() string { return "tlsroutes" }

func (r tlsRouteEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
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
			edges = append(edges, routeBackendsToServices(ctx, route, ns, route.BackendRefs)...)
		}
	}
	return edges
}

func routeBackendsToServices(ctx *framework.Context, route nodes.EdgeNode, routeNS string, backends []nodes.HTTPRouteBackendRef) []model.BloodHoundEdge {
	if ctx == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, backend := range backends {
		backendNS := backend.Namespace
		backendName := backend.Name
		if backendName == "" {
			continue
		}
		if backendNS == "" {
			backendNS = routeNS
		}
		if serviceIndex := ctx.Index.ServicesByNamespace[backendNS]; serviceIndex != nil {
			if svc := serviceIndex[backendName]; svc != nil {
				edges = append(edges, framework.CreateEdge(route, svc, "RoutesTo"))
			}
		}
	}
	return edges
}
