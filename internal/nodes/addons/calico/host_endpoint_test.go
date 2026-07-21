//go:build all_addons

package calico

import "testing"

func TestBuildHostEndpointMapNode(t *testing.T) {
	resource := map[string]any{
		"apiVersion": "crd.projectcalico.org/v1",
		"kind":       "HostEndpoint",
		"metadata": map[string]any{
			"name":   "eth0-worker1",
			"labels": map[string]any{"role": "worker_public"},
		},
		"spec": map[string]any{
			"node":          "worker1",
			"interfaceName": "eth0",
		},
	}

	result, ok := BuildHostEndpointMapNode(resource)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if result.Node.ID != "" {
		t.Fatalf("expected zero-value Node (no graph node), got ID=%q", result.Node.ID)
	}
	if len(result.Core) != 1 {
		t.Fatalf("expected exactly one CoreEntry, got %d", len(result.Core))
	}

	core := result.Core[0].Data.(HostEndpoint)
	if core.Name != "eth0-worker1" {
		t.Fatalf("expected Name=eth0-worker1, got %q", core.Name)
	}
	if core.NodeName != "worker1" {
		t.Fatalf("expected NodeName=worker1, got %q", core.NodeName)
	}
	if core.LabelsMap["role"] != "worker_public" {
		t.Fatalf("expected LabelsMap[role]=worker_public, got %v", core.LabelsMap)
	}
}

func TestBuildHostEndpointMapNode_NoName(t *testing.T) {
	resource := map[string]any{
		"apiVersion": "crd.projectcalico.org/v1",
		"kind":       "HostEndpoint",
		"metadata":   map[string]any{},
		"spec": map[string]any{
			"node": "worker1",
		},
	}

	if _, ok := BuildHostEndpointMapNode(resource); ok {
		t.Fatal("expected ok=false when name is missing")
	}
}
