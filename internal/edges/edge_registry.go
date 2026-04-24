package edges

import (
	"sort"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/edges/rules/addons"
	"bloodhound-kube/internal/edges/rules/mounts"
	"bloodhound-kube/internal/edges/rules/networking"
	"bloodhound-kube/internal/edges/rules/rbac"
	"bloodhound-kube/internal/edges/rules/security"
	"bloodhound-kube/internal/edges/rules/workload"
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
	edges := make([]model.BloodHoundEdge, 0, 256)

	legacyCtx := NewEdgeContext(core)
	for _, rule := range edgeRules {
		results := rule.Apply(legacyCtx)
		if len(results) == 0 {
			continue
		}
		edges = append(edges, results...)
	}

	reg := framework.NewRegistry()
	rbac.Register(reg)
	networking.Register(reg)
	workload.Register(reg)
	security.Register(reg)
	mounts.Register(reg)
	addons.Register(reg)
	ctx := framework.NewContext(core)
	for _, rule := range reg.Rules() {
		results := rule.Apply(ctx)
		if len(results) == 0 {
			continue
		}
		edges = append(edges, results...)
	}

	edges = DeduplicateEdges(edges)
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
