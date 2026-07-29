//go:build no_addons || no_istio

package istio

// Register is a no-op when Istio support isn't compiled in — see istio.go
// for the real implementation.
func Register() {}
