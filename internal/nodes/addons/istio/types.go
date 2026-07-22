package istio

import . "bloodhound-kube/internal/nodes/framework"

// Istio type structs live here (untagged) because internal/model embeds
// them in CoreFacts. The parse/build logic is gated in istio.go
// (//go:build !no_addons && !no_istio).

// SecretRef is a namespaced reference to a Secret, used for Gateway listener
// TLS credentialName resolution.
type SecretRef struct {
	Namespace string
	Name      string
}

// IstioGateway models networking.istio.io Gateway — named to avoid
// collision with the existing Gateway API BHK_Gateway kind.
type IstioGateway struct {
	GraphNodeBase
	SelectorLabels map[string]string
	SecretRefs     []SecretRef
}

type VirtualService struct {
	GraphNodeBase
	GatewayRefs      []SecretRef // reuses {Namespace,Name}; gateways are namespace/name pairs too
	DestinationHosts []string
}

type PeerAuthentication struct {
	GraphNodeBase
	SelectorLabels map[string]string
	MTLSMode       string
}

type AuthorizationPolicy struct {
	GraphNodeBase
	SelectorLabels map[string]string
	Action         string
	Principals     []Principal
}

// Principal is a parsed SPIFFE-style principal string
// ("cluster.local/ns/<namespace>/sa/<name>") from an AuthorizationPolicy
// rule's source.principals list.
type Principal struct {
	Namespace string
	Name      string
}
