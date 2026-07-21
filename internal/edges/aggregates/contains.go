package aggregates

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
)

// Register attaches all aggregates-domain rules to the registry. Called from
// internal/edges/edge_registry.go alongside the other domain Register funcs.
func Register(reg *framework.Registry) {
	reg.Register(aggregateContainsRule{})
}

const containsKind = "BHK_Contains"

var edgePropertiesAggregateContains = map[string]any{
	"Description": "Aggregate node contains this resource (or sub-aggregate).",
}

type aggregateContainsRule struct{}

func (r aggregateContainsRule) Name() string { return "aggregate_contains" }

func (r aggregateContainsRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	edges = append(edges, namespaceAggregateToResources(ctx)...)
	edges = append(edges, clusterAggregateToNamespaceAggregates(ctx)...)
	edges = append(edges, clusterAggregateToClusterResources(ctx)...)
	return edges
}

// namespaceAggregateToResources emits one Contains edge from each per-namespace
// aggregate to every individual resource of that kind in that namespace.
func namespaceAggregateToResources(ctx *framework.Context) []model.BloodHoundEdge {
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}

		if agg := framework.FirstEdgeNode(space.AllPods); agg != nil {
			for i := range space.Pods {
				edges = append(edges, contains(agg, &space.Pods[i]))
			}
		}
		if agg := framework.FirstEdgeNode(space.AllSecrets); agg != nil {
			for i := range space.Secrets {
				edges = append(edges, contains(agg, &space.Secrets[i]))
			}
		}
		if agg := framework.FirstEdgeNode(space.AllConfigMaps); agg != nil {
			for i := range space.ConfigMaps {
				edges = append(edges, contains(agg, &space.ConfigMaps[i]))
			}
		}
		if agg := framework.FirstEdgeNode(space.AllServiceAccounts); agg != nil {
			for i := range space.ServiceAccounts {
				edges = append(edges, contains(agg, &space.ServiceAccounts[i]))
			}
		}
		if agg := framework.FirstEdgeNode(space.AllDeployments); agg != nil {
			for i := range space.Deployments {
				edges = append(edges, contains(agg, &space.Deployments[i]))
			}
		}
		if agg := framework.FirstEdgeNode(space.AllDaemonSets); agg != nil {
			for i := range space.DaemonSets {
				edges = append(edges, contains(agg, &space.DaemonSets[i]))
			}
		}
		if agg := framework.FirstEdgeNode(space.AllStatefulSets); agg != nil {
			for i := range space.StatefulSets {
				edges = append(edges, contains(agg, &space.StatefulSets[i]))
			}
		}
		if agg := framework.FirstEdgeNode(space.AllJobs); agg != nil {
			for i := range space.Jobs {
				edges = append(edges, contains(agg, &space.Jobs[i]))
			}
		}
		if agg := framework.FirstEdgeNode(space.AllCronJobs); agg != nil {
			for i := range space.CronJobs {
				edges = append(edges, contains(agg, &space.CronJobs[i]))
			}
		}
		if agg := framework.FirstEdgeNode(space.AllRoles); agg != nil {
			for i := range space.Roles {
				edges = append(edges, contains(agg, &space.Roles[i]))
			}
		}
	}
	return edges
}

// clusterAggregateToClusterResources emits Contains edges from cluster-scoped
// aggregates that have no namespace counterpart to their individual members.
// Currently covers AllClusterRoles → ClusterRole.
func clusterAggregateToClusterResources(ctx *framework.Context) []model.BloodHoundEdge {
	cluster := ctx.Core.Cluster
	if cluster == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	if cAgg := framework.FirstEdgeNode(cluster.AllClusterRoles); cAgg != nil {
		for i := range cluster.ClusterRoles {
			edges = append(edges, contains(cAgg, &cluster.ClusterRoles[i]))
		}
	}
	return edges
}

// clusterAggregateToNamespaceAggregates emits one Contains edge from each
// cluster aggregate to its corresponding per-namespace aggregate, for every
// discovered namespace. Edges are only emitted when both the cluster and
// per-namespace aggregate are present.
func clusterAggregateToNamespaceAggregates(ctx *framework.Context) []model.BloodHoundEdge {
	cluster := ctx.Core.Cluster
	if cluster == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}

		if cAgg, nsAgg := framework.FirstEdgeNode(cluster.AllPods), framework.FirstEdgeNode(space.AllPods); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := framework.FirstEdgeNode(cluster.AllSecrets), framework.FirstEdgeNode(space.AllSecrets); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := framework.FirstEdgeNode(cluster.AllConfigMaps), framework.FirstEdgeNode(space.AllConfigMaps); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := framework.FirstEdgeNode(cluster.AllServiceAccounts), framework.FirstEdgeNode(space.AllServiceAccounts); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := framework.FirstEdgeNode(cluster.AllDeployments), framework.FirstEdgeNode(space.AllDeployments); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := framework.FirstEdgeNode(cluster.AllDaemonSets), framework.FirstEdgeNode(space.AllDaemonSets); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := framework.FirstEdgeNode(cluster.AllStatefulSets), framework.FirstEdgeNode(space.AllStatefulSets); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := framework.FirstEdgeNode(cluster.AllJobs), framework.FirstEdgeNode(space.AllJobs); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := framework.FirstEdgeNode(cluster.AllCronJobs), framework.FirstEdgeNode(space.AllCronJobs); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
	}
	return edges
}

func contains(start, end nodefw.EdgeNode) model.BloodHoundEdge {
	return framework.CreateEdgeWithProperties(start, end, containsKind, edgePropertiesAggregateContains)
}
