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

		if agg := nsAggregate(space.AllPods); agg != nil {
			for i := range space.Pods {
				edges = append(edges, contains(agg, &space.Pods[i]))
			}
		}
		if agg := nsAggregate(space.AllSecrets); agg != nil {
			for i := range space.Secrets {
				edges = append(edges, contains(agg, &space.Secrets[i]))
			}
		}
		if agg := nsAggregate(space.AllConfigMaps); agg != nil {
			for i := range space.ConfigMaps {
				edges = append(edges, contains(agg, &space.ConfigMaps[i]))
			}
		}
		if agg := nsAggregate(space.AllServiceAccounts); agg != nil {
			for i := range space.ServiceAccounts {
				edges = append(edges, contains(agg, &space.ServiceAccounts[i]))
			}
		}
		if agg := nsAggregate(space.AllDeployments); agg != nil {
			for i := range space.Deployments {
				edges = append(edges, contains(agg, &space.Deployments[i]))
			}
		}
		if agg := nsAggregate(space.AllDaemonSets); agg != nil {
			for i := range space.DaemonSets {
				edges = append(edges, contains(agg, &space.DaemonSets[i]))
			}
		}
		if agg := nsAggregate(space.AllStatefulSets); agg != nil {
			for i := range space.StatefulSets {
				edges = append(edges, contains(agg, &space.StatefulSets[i]))
			}
		}
		if agg := nsAggregate(space.AllJobs); agg != nil {
			for i := range space.Jobs {
				edges = append(edges, contains(agg, &space.Jobs[i]))
			}
		}
		if agg := nsAggregate(space.AllCronJobs); agg != nil {
			for i := range space.CronJobs {
				edges = append(edges, contains(agg, &space.CronJobs[i]))
			}
		}
		if agg := nsAggregate(space.AllRoles); agg != nil {
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
	if cAgg := clusterAggregate(cluster.AllClusterRoles); cAgg != nil {
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

		if cAgg, nsAgg := clusterAggregate(cluster.AllPods), nsAggregate(space.AllPods); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := clusterAggregate(cluster.AllSecrets), nsAggregate(space.AllSecrets); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := clusterAggregate(cluster.AllConfigMaps), nsAggregate(space.AllConfigMaps); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := clusterAggregate(cluster.AllServiceAccounts), nsAggregate(space.AllServiceAccounts); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := clusterAggregate(cluster.AllDeployments), nsAggregate(space.AllDeployments); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := clusterAggregate(cluster.AllDaemonSets), nsAggregate(space.AllDaemonSets); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := clusterAggregate(cluster.AllStatefulSets), nsAggregate(space.AllStatefulSets); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := clusterAggregate(cluster.AllJobs), nsAggregate(space.AllJobs); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
		if cAgg, nsAgg := clusterAggregate(cluster.AllCronJobs), nsAggregate(space.AllCronJobs); cAgg != nil && nsAgg != nil {
			edges = append(edges, contains(cAgg, nsAgg))
		}
	}
	return edges
}

// nsAggregate returns the namespace's aggregate node of the given kind, or
// nil when none is present. Aggregates are emitted exactly once per
// (kind, namespace), so taking [0] is safe.
//
// The returned EdgeNode is a value (not a pointer) because the generic type
// parameter T already satisfies EdgeNode via the embedded GraphNodeBase's
// value-receiver methods. A copy here is cheap (the struct only carries
// metadata used by EdgeID/EdgeKind/etc.).
func nsAggregate[T nodefw.EdgeNode](slice []T) nodefw.EdgeNode {
	if len(slice) == 0 {
		return nil
	}
	return slice[0]
}

// clusterAggregate is the cluster-scoped counterpart of nsAggregate.
func clusterAggregate[T nodefw.EdgeNode](slice []T) nodefw.EdgeNode {
	if len(slice) == 0 {
		return nil
	}
	return slice[0]
}

func contains(start, end nodefw.EdgeNode) model.BloodHoundEdge {
	return framework.CreateEdgeWithProperties(start, end, containsKind, edgePropertiesAggregateContains)
}
