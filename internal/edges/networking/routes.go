package networking

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/networking"
)

type routeNode[P any] interface {
	*P
	nodefw.EdgeNode
	GetBackendRefs() []networking.HTTPRouteBackendRef
}

func applyRouteRule[T any, P routeNode[T]](ctx *framework.Context, getRoutes func(*model.Namespace) []T) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range getRoutes(space) {
			route := P(&getRoutes(space)[i])
			edges = append(edges, routeBackendsToServices(ctx, route, ns, route.GetBackendRefs())...)
		}
	}
	return edges
}

type httpRouteEdgesRule struct{}

func (r httpRouteEdgesRule) Name() string { return "httproutes" }
func (r httpRouteEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	return applyRouteRule[networking.HTTPRoute, *networking.HTTPRoute](ctx, func(s *model.Namespace) []networking.HTTPRoute { return s.HTTPRoutes })
}

type grpcRouteEdgesRule struct{}

func (r grpcRouteEdgesRule) Name() string { return "grpcroutes" }
func (r grpcRouteEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	return applyRouteRule[networking.GRPCRoute, *networking.GRPCRoute](ctx, func(s *model.Namespace) []networking.GRPCRoute { return s.GRPCRoutes })
}

type tcpRouteEdgesRule struct{}

func (r tcpRouteEdgesRule) Name() string { return "tcproutes" }
func (r tcpRouteEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	return applyRouteRule[networking.TCPRoute, *networking.TCPRoute](ctx, func(s *model.Namespace) []networking.TCPRoute { return s.TCPRoutes })
}

type tlsRouteEdgesRule struct{}

func (r tlsRouteEdgesRule) Name() string { return "tlsroutes" }
func (r tlsRouteEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	return applyRouteRule[networking.TLSRoute, *networking.TLSRoute](ctx, func(s *model.Namespace) []networking.TLSRoute { return s.TLSRoutes })
}

func routeBackendsToServices(ctx *framework.Context, route nodefw.EdgeNode, routeNS string, backends []networking.HTTPRouteBackendRef) []model.BloodHoundEdge {
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
				edges = append(edges, framework.CreateEdge(route, svc, "BHK_RoutesTo"))
			}
		}
	}
	return edges
}
