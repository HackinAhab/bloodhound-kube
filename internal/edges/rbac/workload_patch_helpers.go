package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/workload"
)

type namedWorkload[P any] interface {
	*P
	nodefw.EdgeNode
}

func appendNamed[T any, P namedWorkload[T]](ctx *framework.Context, principal nodefw.EdgeNode, names map[string]struct{}, getSlice func(*model.Namespace) []T) []model.BloodHoundEdge {
	if len(names) == 0 {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range getSlice(space) {
			item := P(&getSlice(space)[i])
			if _, ok := names[item.EdgeName()]; ok {
				edges = append(edges, framework.CreateEdgeWithProperties(principal, item, "BHK_WorkloadPatch", edgePropertiesRBACWorkloadPatch))
			}
		}
	}
	return edges
}

func appendNamedPods(ctx *framework.Context, principal nodefw.EdgeNode, names map[string]struct{}) []model.BloodHoundEdge {
	return appendNamed[workload.Pod, *workload.Pod](ctx, principal, names, func(s *model.Namespace) []workload.Pod { return s.Pods })
}

func appendNamedDeployments(ctx *framework.Context, principal nodefw.EdgeNode, names map[string]struct{}) []model.BloodHoundEdge {
	return appendNamed[workload.Deployment, *workload.Deployment](ctx, principal, names, func(s *model.Namespace) []workload.Deployment { return s.Deployments })
}

func appendNamedDaemonSets(ctx *framework.Context, principal nodefw.EdgeNode, names map[string]struct{}) []model.BloodHoundEdge {
	return appendNamed[workload.DaemonSetCore, *workload.DaemonSetCore](ctx, principal, names, func(s *model.Namespace) []workload.DaemonSetCore { return s.DaemonSets })
}

func appendNamedStatefulSets(ctx *framework.Context, principal nodefw.EdgeNode, names map[string]struct{}) []model.BloodHoundEdge {
	return appendNamed[workload.StatefulSetCore, *workload.StatefulSetCore](ctx, principal, names, func(s *model.Namespace) []workload.StatefulSetCore { return s.StatefulSets })
}

func appendNamedJobs(ctx *framework.Context, principal nodefw.EdgeNode, names map[string]struct{}) []model.BloodHoundEdge {
	return appendNamed[workload.Job, *workload.Job](ctx, principal, names, func(s *model.Namespace) []workload.Job { return s.Jobs })
}

func appendNamedCronJobs(ctx *framework.Context, principal nodefw.EdgeNode, names map[string]struct{}) []model.BloodHoundEdge {
	return appendNamed[workload.CronJob, *workload.CronJob](ctx, principal, names, func(s *model.Namespace) []workload.CronJob { return s.CronJobs })
}
