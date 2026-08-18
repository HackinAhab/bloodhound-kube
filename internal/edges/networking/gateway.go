package networking

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/networking"
)

type gatewayEdgesRule struct{}

func (r gatewayEdgesRule) Name() string { return "gateway" }

func (r gatewayEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Gateways {
			gateway := &space.Gateways[i]
			if ctx.Index.External != nil {
				edges = append(edges, framework.CreateEdge(ctx.Index.External, gateway, "BHK_ExternalRoutesTo"))
			}
			for _, routeSpace := range ctx.Core.Namespaces {
				if routeSpace == nil {
					continue
				}
				edges = append(edges, gatewayRouteEdges[networking.HTTPRoute, *networking.HTTPRoute](gateway, ns, routeSpace.HTTPRoutes)...)
				edges = append(edges, gatewayRouteEdges[networking.GRPCRoute, *networking.GRPCRoute](gateway, ns, routeSpace.GRPCRoutes)...)
				edges = append(edges, gatewayRouteEdges[networking.TCPRoute, *networking.TCPRoute](gateway, ns, routeSpace.TCPRoutes)...)
				edges = append(edges, gatewayRouteEdges[networking.TLSRoute, *networking.TLSRoute](gateway, ns, routeSpace.TLSRoutes)...)
			}
		}
	}
	return edges
}

type gatewayRoute[P any] interface {
	*P
	nodefw.EdgeNode
	GetParentGatewayRefs() []nodefw.ParentGatewayRef
}

func gatewayRouteEdges[T any, P gatewayRoute[T]](gw *networking.Gateway, gwNS string, routes []T) []model.BloodHoundEdge {
	var edges []model.BloodHoundEdge
	for i := range routes {
		route := P(&routes[i])
		if routeAttachedToGateway(route.GetParentGatewayRefs(), gw.Name, gwNS) {
			edges = append(edges, framework.CreateEdge(gw, route, "BHK_RoutesTo"))
		}
	}
	return edges
}

func routeAttachedToGateway(parents []nodefw.ParentGatewayRef, gatewayName, gatewayNamespace string) bool {
	for _, parent := range parents {
		if parent.Name == gatewayName && parent.Namespace == gatewayNamespace {
			return true
		}
	}
	return false
}
