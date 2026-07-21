//go:build !all_addons && !calico

package calico

// Register is a no-op when calico support isn't compiled in — see
// global_network_policy.go for the real implementation.
func Register() {}
