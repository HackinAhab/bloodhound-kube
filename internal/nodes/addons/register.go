package addons

import (
	"bloodhound-kube/internal/nodes/addons/calico"
	"bloodhound-kube/internal/nodes/addons/cilium"
	"bloodhound-kube/internal/nodes/addons/externalsecrets"
	"bloodhound-kube/internal/nodes/addons/scc"
)

// Each addon lives in its own subpackage (calico, cilium, externalsecrets,
// scc) for directory clarity. Every subpackage exports Register() — always,
// regardless of build tags: gated subpackages provide a real implementation
// under their tag and a no-op stub otherwise (register_stub.go), mirroring
// this repo's config_embedded.go/config_default.go convention. So this parent
// Register needs no conditionals — SecurityContextConstraints (OpenShift) is
// never gated; the others are no-ops when their tag is absent.
func Register() {
	scc.Register()
	calico.Register()
	cilium.Register()
	externalsecrets.Register()
}
