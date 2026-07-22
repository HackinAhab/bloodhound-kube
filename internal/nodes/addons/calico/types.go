package calico

import . "bloodhound-kube/internal/nodes/framework"

// Calico type structs live here (untagged) because internal/model embeds them in
// CoreFacts, and Go cannot conditionally compile struct fields. The parse/build
// logic that populates them is gated in global_network_policy.go and
// host_endpoint.go (//go:build !no_addons && !no_calico).

type GlobalNetworkPolicy struct {
	GraphNodeBase
	SelectorLabels     map[string]string
	MatchesAll         bool
	SelectorRecognized bool
}

// HostEndpoint represents a Calico HostEndpoint CRD. It is not rendered as
// its own graph node — GlobalNetworkPolicy selectors match a HostEndpoint's
// labels, and the HostEndpoint's spec.node then resolves to the BHK_Node the
// policy actually governs. See BuildHostEndpointMapNode.
type HostEndpoint struct {
	Name      string
	NodeName  string
	LabelsMap map[string]any
}
