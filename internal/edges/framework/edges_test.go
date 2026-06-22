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

func TestDeduplicateEdges_KeyTriple(t *testing.T) {
	// Same start+end but different kind -> two edges.
	// Same start+kind but different end -> two edges.
	// Same end+kind but different start -> two edges.
	edges := []model.BloodHoundEdge{
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "b"}, Kind: "owns"},
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "b"}, Kind: "uses"},
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "c"}, Kind: "owns"},
		{Start: model.BloodHoundEdgeRef{Value: "x"}, End: model.BloodHoundEdgeRef{Value: "b"}, Kind: "owns"},
	}
	unique := DeduplicateEdges(edges)
	if len(unique) != 4 {
		t.Fatalf("expected 4 distinct edges, got %d", len(unique))
	}
}

func TestDeduplicateEdges_Empty(t *testing.T) {
	got := DeduplicateEdges(nil)
	if got == nil {
		t.Fatalf("expected empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected len 0, got %d", len(got))
	}
}

func TestDeduplicateEdges_PreservesProperties(t *testing.T) {
	// First-seen wins; duplicate properties are dropped, not merged.
	edges := []model.BloodHoundEdge{
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "b"}, Kind: "k", Properties: map[string]any{"first": true}},
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "b"}, Kind: "k", Properties: map[string]any{"second": true}},
	}
	unique := DeduplicateEdges(edges)
	if len(unique) != 1 {
		t.Fatalf("expected 1, got %d", len(unique))
	}
	if _, ok := unique[0].Properties["first"]; !ok {
		t.Fatalf("expected first-seen properties to win, got %v", unique[0].Properties)
	}
	if _, ok := unique[0].Properties["second"]; ok {
		t.Fatalf("did not expect second properties to leak in, got %v", unique[0].Properties)
	}
}

func TestSortEdgesStable_OrderingByKindStartEnd(t *testing.T) {
	edges := []model.BloodHoundEdge{
		{Start: model.BloodHoundEdgeRef{Value: "z"}, End: model.BloodHoundEdgeRef{Value: "y"}, Kind: "k1"},
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "b"}, Kind: "k2"},
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "a"}, Kind: "k1"},
		{Start: model.BloodHoundEdgeRef{Value: "z"}, End: model.BloodHoundEdgeRef{Value: "x"}, Kind: "k1"},
	}
	SortEdgesStable(edges)

	expect := []struct{ kind, start, end string }{
		{"k1", "a", "a"},
		{"k1", "z", "x"},
		{"k1", "z", "y"},
		{"k2", "a", "b"},
	}
	for i, want := range expect {
		got := edges[i]
		if got.Kind != want.kind || got.Start.Value != want.start || got.End.Value != want.end {
			t.Fatalf("edge %d: got (%s,%s,%s), want (%s,%s,%s)",
				i, got.Kind, got.Start.Value, got.End.Value, want.kind, want.start, want.end)
		}
	}
}

func TestSortEdgesStable_Idempotent(t *testing.T) {
	edges := []model.BloodHoundEdge{
		{Start: model.BloodHoundEdgeRef{Value: "z"}, End: model.BloodHoundEdgeRef{Value: "y"}, Kind: "k1"},
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "b"}, Kind: "k2"},
		{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "a"}, Kind: "k1"},
	}
	SortEdgesStable(edges)
	first := append([]model.BloodHoundEdge{}, edges...)
	SortEdgesStable(edges)
	for i := range edges {
		if edges[i].Kind != first[i].Kind || edges[i].Start.Value != first[i].Start.Value || edges[i].End.Value != first[i].End.Value {
			t.Fatalf("sort not idempotent at %d", i)
		}
	}
}

// ---------------------------------------------------------------------------
// CreateEdge / CreateEdgeWithProperties
// ---------------------------------------------------------------------------

type stubNode struct {
	id, kind string
}

func (s stubNode) EdgeID() string        { return s.id }
func (s stubNode) EdgeKind() string      { return s.kind }
func (s stubNode) EdgeName() string      { return "" }
func (s stubNode) EdgeNamespace() string { return "" }

func TestCreateEdge_BasicFields(t *testing.T) {
	a := stubNode{id: "Pod:ns:p1", kind: "BHK_Pod"}
	b := stubNode{id: "Node::n1", kind: "BHK_Node"}
	e := CreateEdge(a, b, "BHK_ScheduledOn")
	if e.Kind != "BHK_ScheduledOn" {
		t.Fatalf("kind: got %q", e.Kind)
	}
	if e.Start.Value != "Pod:ns:p1" || e.Start.Kind != "BHK_Pod" || e.Start.MatchBy != "id" {
		t.Fatalf("start: %+v", e.Start)
	}
	if e.End.Value != "Node::n1" || e.End.Kind != "BHK_Node" || e.End.MatchBy != "id" {
		t.Fatalf("end: %+v", e.End)
	}
	if e.Properties != nil {
		t.Fatalf("expected nil properties, got %v", e.Properties)
	}
}

func TestCreateEdgeWithProperties_NilAndEmpty(t *testing.T) {
	a := stubNode{id: "a", kind: "A"}
	b := stubNode{id: "b", kind: "B"}

	if e := CreateEdgeWithProperties(a, b, "k", nil); e.Properties != nil {
		t.Fatalf("nil props should round-trip as nil, got %v", e.Properties)
	}
	if e := CreateEdgeWithProperties(a, b, "k", map[string]any{}); e.Properties != nil {
		t.Fatalf("empty props should be normalized to nil, got %v", e.Properties)
	}
	props := map[string]any{"x": 1}
	e := CreateEdgeWithProperties(a, b, "k", props)
	if e.Properties == nil || e.Properties["x"] != 1 {
		t.Fatalf("expected x=1, got %v", e.Properties)
	}
	// Mutating the input map after creation should not affect the edge.
	props["x"] = 2
	if e.Properties["x"] != 1 {
		t.Fatalf("edge properties should be a copy, got %v", e.Properties)
	}
}

// ---------------------------------------------------------------------------
// ApplyRules
// ---------------------------------------------------------------------------

type fakeRule struct {
	name  string
	apply func(*Context) []model.BloodHoundEdge
}

func (r fakeRule) Name() string                                        { return r.name }
func (r fakeRule) Apply(ctx *Context) []model.BloodHoundEdge { return r.apply(ctx) }

func TestApplyRules_NilContext(t *testing.T) {
	got := ApplyRules(nil, []Rule{fakeRule{name: "r"}})
	if got != nil {
		t.Fatalf("expected nil for nil context, got %v", got)
	}
}

func TestApplyRules_AggregatesAndSkipsEmpty(t *testing.T) {
	rules := []Rule{
		fakeRule{name: "empty", apply: func(*Context) []model.BloodHoundEdge { return nil }},
		fakeRule{name: "one", apply: func(*Context) []model.BloodHoundEdge {
			return []model.BloodHoundEdge{{Start: model.BloodHoundEdgeRef{Value: "a"}, End: model.BloodHoundEdgeRef{Value: "b"}, Kind: "x"}}
		}},
		fakeRule{name: "two", apply: func(*Context) []model.BloodHoundEdge {
			return []model.BloodHoundEdge{
				{Start: model.BloodHoundEdgeRef{Value: "c"}, End: model.BloodHoundEdgeRef{Value: "d"}, Kind: "y"},
				{Start: model.BloodHoundEdgeRef{Value: "e"}, End: model.BloodHoundEdgeRef{Value: "f"}, Kind: "z"},
			}
		}},
	}
	ctx := &Context{}
	got := ApplyRules(ctx, rules)
	if len(got) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(got))
	}
}
