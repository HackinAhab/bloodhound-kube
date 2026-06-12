package edges

import (
	"bloodhound-kube/internal/edges/addons"
	"bloodhound-kube/internal/edges/aggregates"
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/edges/mounts"
	"bloodhound-kube/internal/edges/networking"
	"bloodhound-kube/internal/edges/rbac"
	"bloodhound-kube/internal/edges/security"
	"bloodhound-kube/internal/edges/workload"
	"bloodhound-kube/internal/model"
)

func BuildEdges(core *model.CoreFacts) []model.BloodHoundEdge {
	reg := framework.NewRegistry()
	rbac.Register(reg)
	networking.Register(reg)
	workload.Register(reg)
	security.Register(reg)
	mounts.Register(reg)
	addons.Register(reg)
	aggregates.Register(reg)
	ctx := framework.NewContext(core)
	edges := framework.ApplyRules(ctx, reg.Rules())
	edges = framework.DeduplicateEdges(edges)
	framework.SortEdgesStable(edges)
	return edges
}
