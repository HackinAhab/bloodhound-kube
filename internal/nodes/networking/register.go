package networking

import (
	"bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func Register() {
	framework.RegisterTyped(corev1.SchemeGroupVersion.WithKind("Service"), BuildServiceNode)
	framework.RegisterTyped(networkingv1.SchemeGroupVersion.WithKind("Ingress"), BuildIngressNode)
	framework.RegisterTyped(networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"), BuildNetworkPolicyNode)

	// Gateway API resources are served via CRDs, which default to metadata-only
	// collection (discovery.defaultFetchModeForResource). Their node builders and
	// edge rules read .spec (parentRefs, backendRefs, listeners), so they must be
	// fetched in full or the spec is dropped and no Gateway->Route->Service edges form.
	framework.RegisterTypedWithFetchMode(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("Gateway"), BuildGatewayNode, framework.FetchModeHintFull)
	framework.RegisterTypedWithFetchMode(schema.GroupVersion{Group: gatewayv1beta1.GroupVersion.Group, Version: gatewayv1beta1.GroupVersion.Version}.WithKind("Gateway"), BuildGatewayNode, framework.FetchModeHintFull)

	framework.RegisterTypedWithFetchMode(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("HTTPRoute"), BuildHTTPRouteNode, framework.FetchModeHintFull)
	framework.RegisterTypedWithFetchMode(schema.GroupVersion{Group: gatewayv1beta1.GroupVersion.Group, Version: gatewayv1beta1.GroupVersion.Version}.WithKind("HTTPRoute"), BuildHTTPRouteNode, framework.FetchModeHintFull)

	framework.RegisterTypedWithFetchMode(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("GRPCRoute"), BuildGRPCRouteNode, framework.FetchModeHintFull)
	framework.RegisterTypedWithFetchMode(schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("GRPCRoute"), BuildGRPCRouteNode, framework.FetchModeHintFull)

	framework.RegisterTypedWithFetchMode(schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("TCPRoute"), BuildTCPRouteNode, framework.FetchModeHintFull)

	framework.RegisterTypedWithFetchMode(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("TLSRoute"), BuildTLSRouteNode, framework.FetchModeHintFull)
	framework.RegisterTypedWithFetchMode(schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("TLSRoute"), BuildTLSRouteNode, framework.FetchModeHintFull)
}
