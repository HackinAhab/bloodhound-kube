package framework

import (
	"testing"

	"bloodhound-kube/internal/nodes/workload"
)

func TestLabelsMatchOnly(t *testing.T) {
	cases := []struct {
		name     string
		labels   map[string]any
		selector map[string]string
		want     bool
	}{
		{
			name: "empty selector matches anything",
			labels: map[string]any{"a": "1"},
			selector: nil,
			want:     true,
		},
		{
			name:     "empty selector + nil labels still matches",
			labels:   nil,
			selector: nil,
			want:     true,
		},
		{
			name:     "non-empty selector + nil labels does not match",
			labels:   nil,
			selector: map[string]string{"a": "1"},
			want:     false,
		},
		{
			name:     "exact match",
			labels:   map[string]any{"a": "1", "b": "2"},
			selector: map[string]string{"a": "1"},
			want:     true,
		},
		{
			name:     "value mismatch",
			labels:   map[string]any{"a": "1"},
			selector: map[string]string{"a": "2"},
			want:     false,
		},
		{
			name:     "missing key in labels",
			labels:   map[string]any{"a": "1"},
			selector: map[string]string{"b": "2"},
			want:     false,
		},
		{
			name:     "all keys must match",
			labels:   map[string]any{"a": "1", "b": "2"},
			selector: map[string]string{"a": "1", "b": "2"},
			want:     true,
		},
		{
			name:     "non-string label value (int) does not equal string selector",
			labels:   map[string]any{"a": 1},
			selector: map[string]string{"a": "1"},
			want:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LabelsMatchOnly(tc.labels, tc.selector); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestNormalizeCapability(t *testing.T) {
	cases := map[string]string{
		"":                "",
		"NET_ADMIN":       "BHK_CAP_NET_ADMIN",
		"CAP_SYS_ADMIN":   "BHK_CAP_SYS_ADMIN",
		"sys_ptrace":      "BHK_CAP_sys_ptrace",
		"BHK_CAP_NET_RAW": "BHK_CAP_NET_RAW",
	}
	for in, want := range cases {
		if got := NormalizeCapability(in); got != want {
			t.Errorf("NormalizeCapability(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHasCapability(t *testing.T) {
	pod := workload.Pod{
		CapabilitiesAdd: []string{"NET_ADMIN", "CAP_SYS_ADMIN"},
	}
	cases := []struct {
		cap  string
		want bool
	}{
		{"", false},
		{"CAP_NET_ADMIN", true},   // raw stored as "NET_ADMIN", normalized matches
		{"CAP_SYS_ADMIN", true},
		{"CAP_SYS_PTRACE", false},
	}
	for _, tc := range cases {
		if got := HasCapability(pod, tc.cap); got != tc.want {
			t.Errorf("HasCapability(%q) = %v, want %v", tc.cap, got, tc.want)
		}
	}
}

func TestHostPathMatchesAny(t *testing.T) {
	checks := []string{"/var/lib/kubelet", "/etc"}
	cases := []struct {
		path string
		want bool
	}{
		{"", false},
		{"/var/lib/kubelet", true},          // exact
		{"/var/lib/kubelet/pods", true},     // strict prefix with /
		{"/var/lib/kubeletfoo", false},      // false-prefix sibling does not match
		{"/etc/kubernetes/foo", true},
		{"/etcd", false},                    // sibling
		{"/usr/local/bin", false},
	}
	for _, tc := range cases {
		if got := HostPathMatchesAny(tc.path, checks); got != tc.want {
			t.Errorf("HostPathMatchesAny(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}

	// Empty checks slice yields false for any path.
	if HostPathMatchesAny("/foo", nil) != false {
		t.Errorf("HostPathMatchesAny with nil check list should be false")
	}
}
