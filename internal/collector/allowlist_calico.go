//go:build all_addons || calico

package collector

func init() {
	addonAllowlist = append(addonAllowlist,
		"crd.projectcalico.org/v1/globalnetworkpolicies",
		"crd.projectcalico.org/v1/hostendpoints",
	)
}
