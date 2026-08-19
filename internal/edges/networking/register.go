package networking

import "bloodhound-kube/internal/edges/framework"

func Register(reg *framework.Registry) {
	reg.Register(serviceEdgesRule{})
	reg.Register(serviceRoutesToRule{})
	reg.Register(ingressEdgesRule{})
	reg.Register(routeEdgesRule{})
	reg.Register(gatewayEdgesRule{})
	reg.Register(httpRouteEdgesRule{})
	reg.Register(grpcRouteEdgesRule{})
	reg.Register(tcpRouteEdgesRule{})
	reg.Register(tlsRouteEdgesRule{})
	reg.Register(networkPolicyEdgesRule{})
}
