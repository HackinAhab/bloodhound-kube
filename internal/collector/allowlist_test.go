package collector

import "testing"

func TestParseAllowlistEntries(t *testing.T) {
	entries := []string{
		"v1/pods",
		"apps/v1/deployments",
		"networking.k8s.io",
		"core/*",
		"pods",
	}

	parsed, err := ParseAllowlistEntries(entries)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(parsed) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(parsed))
	}

	if parsed[0].Version != "v1" || parsed[0].Resource != "pods" {
		t.Fatalf("unexpected parsed[0]: %#v", parsed[0])
	}
	if parsed[1].Group != "apps" || parsed[1].Version != "v1" || parsed[1].Resource != "deployments" {
		t.Fatalf("unexpected parsed[1]: %#v", parsed[1])
	}
}

func TestParseAllowlistEntriesInvalid(t *testing.T) {
	if _, err := ParseAllowlistEntries([]string{"a/b/c/d"}); err == nil {
		t.Fatalf("expected error for invalid entry")
	}
}

func TestMergeAllowlistsDedupes(t *testing.T) {
	base := []AllowlistEntry{{Group: "", Version: "v1", Resource: "pods"}}
	extra := []AllowlistEntry{{Group: "", Version: "v1", Resource: "pods"}, {Group: "apps", Version: "v1", Resource: "deployments"}}
	merged := MergeAllowlists(base, extra)
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged entries, got %d", len(merged))
	}
}

func TestFilterDiscoveredResources(t *testing.T) {
	resources := []DiscoveryResource{
		{Group: "", Version: "v1", Resource: "pods"},
		{Group: "apps", Version: "v1", Resource: "deployments"},
	}
	allowlist := []AllowlistEntry{{Group: "apps", Version: "v1"}}

	filtered := FilterDiscoveredResources(resources, allowlist)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered resource, got %d", len(filtered))
	}
	if filtered[0].Resource != "deployments" {
		t.Fatalf("unexpected resource: %#v", filtered[0])
	}
}

func TestIsAPIVersion(t *testing.T) {
	if !isAPIVersion("v1") {
		t.Fatalf("expected v1 to be api version")
	}
	if isAPIVersion("apps") {
		t.Fatalf("expected apps not to be api version")
	}
}
