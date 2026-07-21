//go:build all_addons || cilium

package collector

func init() {
	addonAllowlist = append(addonAllowlist, "cilium.io/v2/ciliumnetworkpolicies")
}
