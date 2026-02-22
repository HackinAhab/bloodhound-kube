package edges

import (
	"sort"

	"bloodhound-kube/internal/model"
)

type EdgeRule interface {
	Name() string
	Apply(ctx *EdgeContext) []model.BloodHoundEdge
}

var edgeRules []EdgeRule

func RegisterEdgeRule(rule EdgeRule) {
	edgeRules = append(edgeRules, rule)
}

func BuildEdges(core *model.CoreFacts) []model.BloodHoundEdge {
	ctx := NewEdgeContext(core)
	edges := make([]model.BloodHoundEdge, 0, 256)
	for _, rule := range edgeRules {
		results := rule.Apply(ctx)
		if len(results) == 0 {
			continue
		}
		edges = append(edges, results...)
	}

	edges = DeduplicateEdges(edges)
	SortEdgesByKind(edges)
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].Kind == edges[j].Kind {
			if edges[i].Start.Value == edges[j].Start.Value {
				return edges[i].End.Value < edges[j].End.Value
			}
			return edges[i].Start.Value < edges[j].Start.Value
		}
		return edges[i].Kind < edges[j].Kind
	})
	return edges
}
