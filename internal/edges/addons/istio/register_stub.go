//go:build no_addons || no_istio

package istio

import "bloodhound-kube/internal/edges/framework"

// Register is a no-op when Istio support isn't compiled in — see istio.go
// for the real implementation.
func Register(reg *framework.Registry) {}
