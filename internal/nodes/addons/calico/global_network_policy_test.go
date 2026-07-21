//go:build all_addons

package calico

import (
	"reflect"
	"testing"
)

func TestParseCalicoSelector(t *testing.T) {
	cases := []struct {
		name           string
		expr           string
		wantLabels     map[string]string
		wantMatchesAll bool
		wantOk         bool
	}{
		{"empty", "", map[string]string{}, true, true},
		{"all", "all()", map[string]string{}, true, true},
		{"single equality", "role == 'frontend'", map[string]string{"role": "frontend"}, false, true},
		{"and equality", "role == 'frontend' && env == 'prod'", map[string]string{"role": "frontend", "env": "prod"}, false, true},
		{"comma equality", "role == 'frontend', env == 'prod'", map[string]string{"role": "frontend", "env": "prod"}, false, true},
		{"has unsupported", "has(role)", nil, false, false},
		{"in unsupported", "role in {'frontend','backend'}", nil, false, false},
		{"or unsupported", "role == 'frontend' || role == 'backend'", nil, false, false},
		{"negation unsupported", "!has(role)", nil, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			labels, matchesAll, ok := parseCalicoSelector(tc.expr)
			if ok != tc.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOk)
			}
			if ok && (matchesAll != tc.wantMatchesAll || !reflect.DeepEqual(labels, tc.wantLabels)) {
				t.Fatalf("got (labels=%v, matchesAll=%v), want (labels=%v, matchesAll=%v)", labels, matchesAll, tc.wantLabels, tc.wantMatchesAll)
			}
		})
	}
}

func TestBuildGlobalNetworkPolicyMapNode_MatchesAll(t *testing.T) {
	resource := map[string]any{
		"apiVersion": "crd.projectcalico.org/v1",
		"kind":       "GlobalNetworkPolicy",
		"metadata":   map[string]any{"name": "deny-all"},
		"spec": map[string]any{
			"selector": "all()",
			"types":    []any{"Ingress"},
		},
	}

	result, ok := BuildGlobalNetworkPolicyMapNode(resource)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if result.Node.ID != "BHK_GlobalNetworkPolicy:deny-all" {
		t.Fatalf("unexpected node ID: %s", result.Node.ID)
	}
	core := result.Core[0].Data.(GlobalNetworkPolicy)
	if !core.MatchesAll || !core.SelectorRecognized {
		t.Fatalf("expected MatchesAll=true, SelectorRecognized=true, got %+v", core)
	}
	if !result.Core[0].Cluster {
		t.Fatal("expected cluster-scoped CoreEntry")
	}
}

func TestBuildGlobalNetworkPolicyMapNode_LabelSelector(t *testing.T) {
	resource := map[string]any{
		"apiVersion": "crd.projectcalico.org/v1",
		"kind":       "GlobalNetworkPolicy",
		"metadata":   map[string]any{"name": "web-policy"},
		"spec": map[string]any{
			"selector": "role == 'frontend'",
		},
	}

	result, ok := BuildGlobalNetworkPolicyMapNode(resource)
	if !ok {
		t.Fatal("expected ok=true")
	}
	core := result.Core[0].Data.(GlobalNetworkPolicy)
	if core.MatchesAll || !core.SelectorRecognized {
		t.Fatalf("expected MatchesAll=false, SelectorRecognized=true, got %+v", core)
	}
	if core.SelectorLabels["role"] != "frontend" {
		t.Fatalf("expected SelectorLabels[role]=frontend, got %v", core.SelectorLabels)
	}
}

func TestBuildGlobalNetworkPolicyMapNode_UnrecognizedSelector(t *testing.T) {
	resource := map[string]any{
		"apiVersion": "crd.projectcalico.org/v1",
		"kind":       "GlobalNetworkPolicy",
		"metadata":   map[string]any{"name": "has-policy"},
		"spec": map[string]any{
			"selector": "has(role)",
		},
	}

	result, ok := BuildGlobalNetworkPolicyMapNode(resource)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if result.Node.Properties["selector"] != "has(role)" {
		t.Fatalf("expected raw selector to remain visible in properties, got %v", result.Node.Properties["selector"])
	}
	core := result.Core[0].Data.(GlobalNetworkPolicy)
	if core.SelectorRecognized {
		t.Fatal("expected SelectorRecognized=false for unsupported selector syntax")
	}
}

func TestBuildGlobalNetworkPolicyMapNode_NoName(t *testing.T) {
	resource := map[string]any{
		"apiVersion": "crd.projectcalico.org/v1",
		"kind":       "GlobalNetworkPolicy",
		"metadata":   map[string]any{},
	}
	if _, ok := BuildGlobalNetworkPolicyMapNode(resource); ok {
		t.Fatal("expected ok=false for policy with no name")
	}
}
