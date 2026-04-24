package framework

import (
	"maps"

	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
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

func CreateEdge(start, end nodes.EdgeNode, kind string) model.BloodHoundEdge {
	return CreateEdgeWithProperties(start, end, kind, nil)
}

func CreateEdgeWithProperties(start, end nodes.EdgeNode, kind string, props map[string]any) model.BloodHoundEdge {
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
