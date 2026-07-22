//go:build !no_addons && !no_cilium

package collector

func init() {
	addonAllowlist = append(addonAllowlist, "cilium.io/v2/ciliumnetworkpolicies")
}
