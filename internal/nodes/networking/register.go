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
	framework.RegisterResources(
		framework.ResourceRegistration{
			GVK:          corev1.SchemeGroupVersion.WithKind("Service"),
			TypedBuilder: BuildServiceNode,
		},
		framework.ResourceRegistration{
			GVK:          networkingv1.SchemeGroupVersion.WithKind("Ingress"),
			TypedBuilder: BuildIngressNode,
		},
		framework.ResourceRegistration{
			GVK:          networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"),
			TypedBuilder: BuildNetworkPolicyNode,
		},

		framework.ResourceRegistration{
			GVK:          schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("Gateway"),
			TypedBuilder: BuildGatewayNode,
			FetchMode:    framework.FetchModeHintFull,
		},
		framework.ResourceRegistration{
			GVK:          schema.GroupVersion{Group: gatewayv1beta1.GroupVersion.Group, Version: gatewayv1beta1.GroupVersion.Version}.WithKind("Gateway"),
			TypedBuilder: BuildGatewayNode,
			FetchMode:    framework.FetchModeHintFull,
		},
		framework.ResourceRegistration{
			GVK:          schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("HTTPRoute"),
			TypedBuilder: BuildHTTPRouteNode,
			FetchMode:    framework.FetchModeHintFull,
		},
		framework.ResourceRegistration{
			GVK:          schema.GroupVersion{Group: gatewayv1beta1.GroupVersion.Group, Version: gatewayv1beta1.GroupVersion.Version}.WithKind("HTTPRoute"),
			TypedBuilder: BuildHTTPRouteNode,
			FetchMode:    framework.FetchModeHintFull,
		},
		framework.ResourceRegistration{
			GVK:          schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("GRPCRoute"),
			TypedBuilder: BuildGRPCRouteNode,
			FetchMode:    framework.FetchModeHintFull,
		},
		framework.ResourceRegistration{
			GVK:          schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("GRPCRoute"),
			TypedBuilder: BuildGRPCRouteNode,
			FetchMode:    framework.FetchModeHintFull,
		},
		framework.ResourceRegistration{
			GVK:          schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("TCPRoute"),
			TypedBuilder: BuildTCPRouteNode,
			FetchMode:    framework.FetchModeHintFull,
		},
		framework.ResourceRegistration{
			GVK:          schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("TLSRoute"),
			TypedBuilder: BuildTLSRouteNode,
			FetchMode:    framework.FetchModeHintFull},
		framework.ResourceRegistration{
			GVK:          schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("TLSRoute"),
			TypedBuilder: BuildTLSRouteNode,
			FetchMode:    framework.FetchModeHintFull,
		},
	)
	framework.RegisterDefaultCollections(
		framework.CollectionTarget{
			Version:  "v1",
			Resource: "services",
		},
		framework.CollectionTarget{
			Group:    "networking.k8s.io",
			Version:  "v1",
			Resource: "ingresses",
		},
		framework.CollectionTarget{
			Group:    "networking.k8s.io",
			Version:  "v1",
			Resource: "networkpolicies",
		},
		framework.CollectionTarget{
			Group:   gatewayv1.GroupVersion.Group,
			Version: gatewayv1.GroupVersion.Version,
		},
		framework.CollectionTarget{
			Group:    gatewayv1beta1.GroupVersion.Group,
			Version:  gatewayv1beta1.GroupVersion.Version,
			Resource: "gateways",
		},
		framework.CollectionTarget{
			Group:    gatewayv1alpha2.GroupVersion.Group,
			Version:  gatewayv1alpha2.GroupVersion.Version,
			Resource: "grpcroutes",
		},
		framework.CollectionTarget{
			Group:    gatewayv1alpha2.GroupVersion.Group,
			Version:  gatewayv1alpha2.GroupVersion.Version,
			Resource: "tcproutes",
		},
		framework.CollectionTarget{
			Group:    gatewayv1alpha2.GroupVersion.Group,
			Version:  gatewayv1alpha2.GroupVersion.Version,
			Resource: "tlsroutes",
		},
	)
}
