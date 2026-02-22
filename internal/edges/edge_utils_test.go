package edges

import (
	"testing"

	"bloodhound-kube/internal/model"
)

func TestEdgeHelpers(t *testing.T) {
	edges := []model.BloodHoundEdge{
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "b"}, Kind: "uses"},
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "b"}, Kind: "uses"},
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "c"}, Kind: "owns"},
	}

	unique := DeduplicateEdges(edges)
	if len(unique) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(unique))
	}

	SortEdgesByKind(unique)
	if unique[0].Kind != "owns" {
		t.Fatalf("expected first kind owns, got %s", unique[0].Kind)
	}

	stats := GetEdgeStats(unique)
	if stats["owns"] != 1 || stats["uses"] != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}
