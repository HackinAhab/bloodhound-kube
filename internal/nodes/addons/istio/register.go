//go:build !no_addons && !no_istio

package istio

import (
	. "bloodhound-kube/internal/nodes/framework"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Register() {
	RegisterResources(
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1", Kind: "Gateway"},
			MapBuilder: BuildIstioGatewayNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1", Kind: "VirtualService"},
			MapBuilder: BuildVirtualServiceNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "security.istio.io", Version: "v1", Kind: "PeerAuthentication"},
			MapBuilder: BuildPeerAuthenticationNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "security.istio.io", Version: "v1", Kind: "AuthorizationPolicy"},
			MapBuilder: BuildAuthorizationPolicyNode,
			FetchMode:  FetchModeHintFull,
		},
	)
	RegisterDefaultCollections(
		CollectionTarget{
			Group:    "networking.istio.io",
			Version:  "v1",
			Resource: "gateways",
		},
		CollectionTarget{
			Group:    "networking.istio.io",
			Version:  "v1",
			Resource: "virtualservices",
		},
		CollectionTarget{
			Group:    "security.istio.io",
			Version:  "v1",
			Resource: "peerauthentications",
		},
		CollectionTarget{
			Group:    "security.istio.io",
			Version:  "v1",
			Resource: "authorizationpolicies",
		},
	)
}
