//go:build !no_addons && !no_istio

package collector

func init() {
	addonAllowlist = append(addonAllowlist,
		"networking.istio.io/v1/gateways",
		"networking.istio.io/v1/virtualservices",
		"security.istio.io/v1/peerauthentications",
		"security.istio.io/v1/authorizationpolicies",
	)
}
