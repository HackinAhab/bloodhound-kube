package cilium

import . "bloodhound-kube/internal/nodes/framework"

// CiliumNetworkPolicy lives here (untagged) because internal/model embeds it in
// CoreFacts. The parse/build logic is gated in cilium_network_policy.go
// (//go:build !no_addons && !no_cilium).
type CiliumNetworkPolicy struct {
	GraphNodeBase
	PodSelectorLabels map[string]string
}
