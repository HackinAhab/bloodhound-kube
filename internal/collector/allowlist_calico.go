//go:build !no_addons && !no_calico

package collector

func init() {
	addonAllowlist = append(addonAllowlist,
		"crd.projectcalico.org/v1/globalnetworkpolicies",
		"crd.projectcalico.org/v1/hostendpoints",
	)
}
