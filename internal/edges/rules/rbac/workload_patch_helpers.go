package rbac

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

func appendNamedPods(ctx *framework.Context, sa *nodes.ServiceAccount, names map[string]struct{}) []model.BloodHoundEdge {
	if len(names) == 0 {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Pods {
			pod := &space.Pods[i]
			if _, ok := names[pod.Name]; ok {
				edges = append(edges, framework.CreateEdgeWithProperties(sa, pod, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
			}
		}
	}
	return edges
}

func appendNamedDeployments(ctx *framework.Context, sa *nodes.ServiceAccount, names map[string]struct{}) []model.BloodHoundEdge {
	if len(names) == 0 {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Deployments {
			deployment := &space.Deployments[i]
			if _, ok := names[deployment.Name]; ok {
				edges = append(edges, framework.CreateEdgeWithProperties(sa, deployment, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
			}
		}
	}
	return edges
}

func appendNamedDaemonSets(ctx *framework.Context, sa *nodes.ServiceAccount, names map[string]struct{}) []model.BloodHoundEdge {
	if len(names) == 0 {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.DaemonSets {
			daemonSet := &space.DaemonSets[i]
			if _, ok := names[daemonSet.Name]; ok {
				edges = append(edges, framework.CreateEdgeWithProperties(sa, daemonSet, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
			}
		}
	}
	return edges
}

func appendNamedStatefulSets(ctx *framework.Context, sa *nodes.ServiceAccount, names map[string]struct{}) []model.BloodHoundEdge {
	if len(names) == 0 {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.StatefulSets {
			statefulSet := &space.StatefulSets[i]
			if _, ok := names[statefulSet.Name]; ok {
				edges = append(edges, framework.CreateEdgeWithProperties(sa, statefulSet, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
			}
		}
	}
	return edges
}

func appendNamedJobs(ctx *framework.Context, sa *nodes.ServiceAccount, names map[string]struct{}) []model.BloodHoundEdge {
	if len(names) == 0 {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Jobs {
			job := &space.Jobs[i]
			if _, ok := names[job.Name]; ok {
				edges = append(edges, framework.CreateEdgeWithProperties(sa, job, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
			}
		}
	}
	return edges
}

func appendNamedCronJobs(ctx *framework.Context, sa *nodes.ServiceAccount, names map[string]struct{}) []model.BloodHoundEdge {
	if len(names) == 0 {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.CronJobs {
			cronJob := &space.CronJobs[i]
			if _, ok := names[cronJob.Name]; ok {
				edges = append(edges, framework.CreateEdgeWithProperties(sa, cronJob, "WorkloadPatch", edgePropertiesRBACWorkloadPatch))
			}
		}
	}
	return edges
}
