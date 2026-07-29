//go:build no_addons || no_cilium

package cilium

// Register is a no-op when cilium support isn't compiled in — see
// cilium_network_policy.go for the real implementation.
func Register() {}
