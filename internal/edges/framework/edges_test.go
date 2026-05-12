package framework

import (
	"testing"

	"bloodhound-kube/internal/model"
)

func TestDeduplicateAndSortEdges(t *testing.T) {
	edges := []model.BloodHoundEdge{
		{Start: model.BloodHoundEdgeRef{Value: "z"}, End: model.BloodHoundEdgeRef{Value: "a"}, Kind: "uses"},
		{Start: model.BloodHoundEdgeRef{Value: "z"}, End: model.BloodHoundEdgeRef{Value: "a"}, Kind: "uses"},
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "c"}, Kind: "owns"},
	}

	unique := DeduplicateEdges(edges)
	if len(unique) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(unique))
	}

	SortEdgesStable(unique)
	if unique[0].Kind != "owns" {
		t.Fatalf("expected first kind owns, got %s", unique[0].Kind)
	}
}
