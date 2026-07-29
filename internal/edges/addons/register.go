package addons

import (
	"bloodhound-kube/internal/edges/addons/calico"
	"bloodhound-kube/internal/edges/addons/certmanager"
	"bloodhound-kube/internal/edges/addons/cilium"
	"bloodhound-kube/internal/edges/addons/externalsecrets"
	"bloodhound-kube/internal/edges/addons/istio"
	"bloodhound-kube/internal/edges/framework"
)

// Each addon's edge rules live in their own subpackage, mirroring the nodes
// side. Every subpackage exports Register(reg) unconditionally (real
// implementation under its build tag, no-op stub otherwise — see
// register_stub.go in each), so this parent needs no conditionals.
func Register(reg *framework.Registry) {
	calico.Register(reg)
	cilium.Register(reg)
	externalsecrets.Register(reg)
	certmanager.Register(reg)
	istio.Register(reg)
}
