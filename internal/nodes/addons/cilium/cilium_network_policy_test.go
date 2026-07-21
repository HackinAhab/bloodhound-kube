//go:build all_addons

package cilium

import (
	"testing"
)

func TestBuildCiliumNetworkPolicyNode_BasicProperties(t *testing.T) {
	resource := map[string]any{
		"metadata": map[string]any{
			"name":      "allow-web",
			"namespace": "prod",
			"labels":    map[string]any{"team": "platform"},
		},
		"spec": map[string]any{
			"endpointSelector": map[string]any{
				"matchLabels": map[string]any{"app": "web"},
			},
			"ingress": []any{
				map[string]any{
					"fromEndpoints": []any{
						map[string]any{"matchLabels": map[string]any{"role": "frontend"}},
					},
				},
			},
			"egress": []any{
				map[string]any{
					"toCIDR": []any{"10.0.0.0/8"},
				},
			},
		},
	}

	result, ok := BuildCiliumNetworkPolicyNode(resource)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if result.Node.ID != "BHK_CiliumNetworkPolicy:prod:allow-web" {
		t.Fatalf("unexpected node ID: %s", result.Node.ID)
	}
	if got := result.Node.Properties["ingress"].([]string); len(got) != 1 {
		t.Fatalf("ingress = %v, want 1 rule", got)
	}
	if got := result.Node.Properties["egress"].([]string); len(got) != 1 {
		t.Fatalf("egress = %v, want 1 rule", got)
	}
	core := result.Core[0].Data.(CiliumNetworkPolicy)
	if core.PodSelectorLabels["app"] != "web" {
		t.Fatalf("expected PodSelectorLabels[app]=web, got %v", core.PodSelectorLabels)
	}
	if result.Core[0].Namespace != "prod" || result.Core[0].Cluster {
		t.Fatalf("expected namespaced CoreEntry for prod, got %+v", result.Core[0])
	}
}

func TestBuildCiliumNetworkPolicyNode_EmptySelector(t *testing.T) {
	resource := map[string]any{
		"metadata": map[string]any{"name": "deny-all", "namespace": "ns1"},
		"spec":     map[string]any{},
	}

	result, ok := BuildCiliumNetworkPolicyNode(resource)
	if !ok {
		t.Fatal("expected ok=true")
	}
	sel := result.Node.Properties["endpointSelector"].([]string)
	if len(sel) != 1 || sel[0] != "<all endpoints>" {
		t.Fatalf("endpointSelector = %v, want [\"<all endpoints>\"]", sel)
	}
	core := result.Core[0].Data.(CiliumNetworkPolicy)
	if len(core.PodSelectorLabels) != 0 {
		t.Fatalf("expected empty PodSelectorLabels, got %v", core.PodSelectorLabels)
	}
}

func TestBuildCiliumNetworkPolicyNode_NoName(t *testing.T) {
	resource := map[string]any{"metadata": map[string]any{}}
	if _, ok := BuildCiliumNetworkPolicyNode(resource); ok {
		t.Fatal("expected ok=false for resource with no name")
	}
}

func TestSummarizeCiliumSelector_MatchExpressions(t *testing.T) {
	sel := map[string]any{
		"matchExpressions": []any{
			map[string]any{"key": "env", "operator": "In", "values": []any{"prod", "staging"}},
		},
	}
	got := summarizeCiliumSelector(sel)
	if len(got) != 1 || got[0] != "env In [prod,staging]" {
		t.Fatalf("summarizeCiliumSelector = %v, want [\"env In [prod,staging]\"]", got)
	}
}
