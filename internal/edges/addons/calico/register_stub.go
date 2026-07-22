//go:build no_addons || no_calico

package calico

import "bloodhound-kube/internal/edges/framework"

// Register is a no-op when calico support isn't compiled in — see
// global_network_policy.go for the real implementation.
func Register(reg *framework.Registry) {}
