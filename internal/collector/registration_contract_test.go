//go:build !no_addons && !no_calico && !no_cilium && !no_cert_manager && !no_istio && !no_external_secrets

package collector

import "testing"

func TestDefaultDiscoveryAllowlistIncludesDescriptorResources(t *testing.T) {
	allowlist, err := DefaultDiscoveryAllowlist()
	if err != nil {
		t.Fatalf("DefaultDiscoveryAllowlist() error = %v", err)
	}

	want := []AllowlistEntry{
		{Group: "gateway.networking.k8s.io", Version: "v1"},
		{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Resource: "grpcroutes"},
		{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "gateways"},
		{Group: "cert-manager.io", Version: "v1", Resource: "certificates"},
		{Group: "networking.istio.io", Version: "v1", Resource: "virtualservices"},
		{Group: "cilium.io", Version: "v2", Resource: "ciliumnetworkpolicies"},
		{Group: "projectcalico.org", Version: "v3", Resource: "hostendpoints"},
		{Group: "route.openshift.io", Version: "v1", Resource: "routes"},
	}
	for _, entry := range want {
		if !containsAllowlistEntry(allowlist, entry) {
			t.Errorf("default allowlist missing descriptor resource %#v", entry)
		}
	}
}

func TestDescriptorCRDsUseFullFetch(t *testing.T) {
	resources := []DiscoveryResource{
		{Group: "cert-manager.io", Version: "v1", Kind: "Certificate", IsCRD: true},
		{Group: "networking.istio.io", Version: "v1", Kind: "VirtualService", IsCRD: true},
		{Group: "external-secrets.io", Version: "v1", Kind: "ExternalSecret", IsCRD: true},
		{Group: "external-secrets.io", Version: "v1beta1", Kind: "SecretStore", IsCRD: true},
	}
	for _, resource := range resources {
		if got := defaultFetchModeForResource(resource); got != FetchModeFull {
			t.Errorf("%s/%s %s fetch mode = %q, want %q", resource.Group, resource.Version, resource.Kind, got, FetchModeFull)
		}
	}
}

func containsAllowlistEntry(entries []AllowlistEntry, want AllowlistEntry) bool {
	for _, entry := range entries {
		if entry == want {
			return true
		}
	}
	return false
}
