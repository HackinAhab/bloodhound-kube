package networking

import (
	"bloodhound-kube/internal/nodes/framework"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func Register(reg *framework.Registry) {
	reg.RegisterTyped(networkingv1.SchemeGroupVersion.WithKind("Service"), BuildServiceNode)
	reg.RegisterTyped(networkingv1.SchemeGroupVersion.WithKind("Ingress"), BuildIngressNode)
	reg.RegisterTyped(networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"), BuildNetworkPolicyNode)

	reg.RegisterTyped(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("Gateway"), BuildGatewayNode)
	reg.RegisterTyped(schema.GroupVersion{Group: gatewayv1beta1.GroupVersion.Group, Version: gatewayv1beta1.GroupVersion.Version}.WithKind("Gateway"), BuildGatewayNode)

	reg.RegisterTyped(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("HTTPRoute"), BuildHTTPRouteNode)
	reg.RegisterTyped(schema.GroupVersion{Group: gatewayv1beta1.GroupVersion.Group, Version: gatewayv1beta1.GroupVersion.Version}.WithKind("HTTPRoute"), BuildHTTPRouteNode)

	reg.RegisterTyped(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("GRPCRoute"), BuildGRPCRouteNode)
	reg.RegisterTyped(schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("GRPCRoute"), BuildGRPCRouteNode)

	reg.RegisterTyped(schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("TCPRoute"), BuildTCPRouteNode)

	reg.RegisterTyped(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("TLSRoute"), BuildTLSRouteNode)
	reg.RegisterTyped(schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("TLSRoute"), BuildTLSRouteNode)
}
