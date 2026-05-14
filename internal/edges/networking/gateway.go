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
				edges = append(edges, framework.CreateEdge(gateway, ctx.Index.External, "ExternalRoutesTo"))
			}
			edges = append(edges, gatewayHTTPRouteEdges(ctx, gateway, ns)...)
			edges = append(edges, gatewayGRPCRouteEdges(ctx, gateway, ns)...)
			edges = append(edges, gatewayTCPRouteEdges(ctx, gateway, ns)...)
			edges = append(edges, gatewayTLSRouteEdges(ctx, gateway, ns)...)
		}
	}
	return edges
}

func gatewayHTTPRouteEdges(ctx *framework.Context, gatewayRef *networking.Gateway, gatewayNS string) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	edges := []model.BloodHoundEdge{}
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.HTTPRoutes {
			route := &space.HTTPRoutes[i]
			if routeAttachedToGateway(route.ParentGatewayRefs, gatewayRef.Name, gatewayNS) {
				edges = append(edges, framework.CreateEdge(gatewayRef, route, "RoutesTo"))
			}
		}
	}
	return edges
}

func gatewayGRPCRouteEdges(ctx *framework.Context, gatewayRef *networking.Gateway, gatewayNS string) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	edges := []model.BloodHoundEdge{}
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.GRPCRoutes {
			route := &space.GRPCRoutes[i]
			if routeAttachedToGateway(route.ParentGatewayRefs, gatewayRef.Name, gatewayNS) {
				edges = append(edges, framework.CreateEdge(gatewayRef, route, "RoutesTo"))
			}
		}
	}
	return edges
}

func gatewayTCPRouteEdges(ctx *framework.Context, gatewayRef *networking.Gateway, gatewayNS string) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	edges := []model.BloodHoundEdge{}
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.TCPRoutes {
			route := &space.TCPRoutes[i]
			if routeAttachedToGateway(route.ParentGatewayRefs, gatewayRef.Name, gatewayNS) {
				edges = append(edges, framework.CreateEdge(gatewayRef, route, "RoutesTo"))
			}
		}
	}
	return edges
}

func gatewayTLSRouteEdges(ctx *framework.Context, gatewayRef *networking.Gateway, gatewayNS string) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	edges := []model.BloodHoundEdge{}
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.TLSRoutes {
			route := &space.TLSRoutes[i]
			if routeAttachedToGateway(route.ParentGatewayRefs, gatewayRef.Name, gatewayNS) {
				edges = append(edges, framework.CreateEdge(gatewayRef, route, "RoutesTo"))
			}
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
