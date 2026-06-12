package framework

import (
	"maps"
	"sort"

	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
)

func ApplyRules(ctx *Context, rules []Rule) []model.BloodHoundEdge {
	if ctx == nil {
		return nil
	}
	edges := make([]model.BloodHoundEdge, 0, 256)
	for _, rule := range rules {
		results := rule.Apply(ctx)
		if len(results) == 0 {
			continue
		}
		edges = append(edges, results...)
	}
	return edges
}

func CreateEdge(start, end nodefw.EdgeNode, kind string) model.BloodHoundEdge {
	return CreateEdgeWithProperties(start, end, kind, nil)
}

func CreateEdgeWithProperties(start, end nodefw.EdgeNode, kind string, props map[string]any) model.BloodHoundEdge {
	properties := map[string]any{}
	maps.Copy(properties, props)
	if len(properties) == 0 {
		properties = nil
	}
	return model.BloodHoundEdge{
		Start: model.BloodHoundEdgeRef{
			MatchBy: "id",
			Value:   start.EdgeID(),
			Kind:    start.EdgeKind(),
		},
		End: model.BloodHoundEdgeRef{
			MatchBy: "id",
			Value:   end.EdgeID(),
			Kind:    end.EdgeKind(),
		},
		Kind:       kind,
		Properties: properties,
	}
}

func DeduplicateEdges(edges []model.BloodHoundEdge) []model.BloodHoundEdge {
	type edgeKey struct {
		start string
		end   string
		kind  string
	}

	seen := make(map[edgeKey]struct{}, len(edges))
	unique := make([]model.BloodHoundEdge, 0, len(edges))
	for _, edge := range edges {
		key := edgeKey{start: edge.Start.Value, end: edge.End.Value, kind: edge.Kind}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, edge)
	}
	return unique
}

func SortEdgesStable(edges []model.BloodHoundEdge) {
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].Kind == edges[j].Kind {
			if edges[i].Start.Value == edges[j].Start.Value {
				return edges[i].End.Value < edges[j].End.Value
			}
			return edges[i].Start.Value < edges[j].Start.Value
		}
		return edges[i].Kind < edges[j].Kind
	})
}
