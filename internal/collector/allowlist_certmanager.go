//go:build !no_addons && !no_cert_manager

package collector

func init() {
	addonAllowlist = append(addonAllowlist,
		"cert-manager.io/v1/certificates",
		"cert-manager.io/v1/issuers",
		"cert-manager.io/v1/clusterissuers",
	)
}
