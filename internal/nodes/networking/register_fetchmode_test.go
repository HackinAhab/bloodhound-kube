package networking

import (
	"testing"

	"bloodhound-kube/internal/nodes/framework"

	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// Gateway API resources are served via CRDs. discovery.defaultFetchModeForResource
// collects CRDs as metadata-only unless a builder registers FetchModeHintFull.
// These types' builders and edge rules read .spec (parentRefs/backendRefs/listeners),
// so a missing full hint silently drops the spec and no Gateway->Route->Service
// edges form. Guard against regressing back to the metadata default.
func TestGatewayAPITypesRegisterFullFetchMode(t *testing.T) {
	Register()

	gvks := []schema.GroupVersionKind{
		{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version, Kind: "Gateway"},
		{Group: gatewayv1beta1.GroupVersion.Group, Version: gatewayv1beta1.GroupVersion.Version, Kind: "Gateway"},
		{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version, Kind: "HTTPRoute"},
		{Group: gatewayv1beta1.GroupVersion.Group, Version: gatewayv1beta1.GroupVersion.Version, Kind: "HTTPRoute"},
		{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version, Kind: "GRPCRoute"},
		{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version, Kind: "GRPCRoute"},
		{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version, Kind: "TCPRoute"},
		{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version, Kind: "TLSRoute"},
		{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version, Kind: "TLSRoute"},
	}

	for _, gvk := range gvks {
		mode, ok := framework.TypedFetchModeHint(gvk)
		if !ok {
			t.Errorf("%s: no fetch-mode hint registered (would default to metadata-only for CRDs, dropping .spec)", gvk)
			continue
		}
		if mode != framework.FetchModeHintFull {
			t.Errorf("%s: fetch-mode hint = %q, want %q", gvk, mode, framework.FetchModeHintFull)
		}
	}
}
