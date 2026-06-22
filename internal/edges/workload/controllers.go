package workload

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type deploymentEdgesRule struct{}

func (r deploymentEdgesRule) Name() string { return "deployment" }

func (r deploymentEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Deployments {
			deploy := &space.Deployments[i]
			if len(deploy.SelectorLabels) == 0 {
				continue
			}
			for j := range space.Pods {
				pod := &space.Pods[j]
				if framework.LabelsMatchOnly(pod.LabelsMap, deploy.SelectorLabels) {
					edges = append(edges, framework.CreateEdge(deploy, pod, "BHK_ManagedBy"))
				}
			}
		}
	}
	return edges
}

type daemonSetEdgesRule struct{}

func (r daemonSetEdgesRule) Name() string { return "daemonset" }

func (r daemonSetEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.DaemonSets {
			daemonSet := &space.DaemonSets[i]
			if len(daemonSet.SelectorLabels) == 0 {
				continue
			}
			for j := range space.Pods {
				pod := &space.Pods[j]
				if framework.LabelsMatchOnly(pod.LabelsMap, daemonSet.SelectorLabels) {
					edges = append(edges, framework.CreateEdge(daemonSet, pod, "BHK_ManagedBy"))
				}
			}
		}
	}
	return edges
}

type statefulSetEdgesRule struct{}

func (r statefulSetEdgesRule) Name() string { return "statefulset" }

func (r statefulSetEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.StatefulSets {
			statefulSet := &space.StatefulSets[i]
			if len(statefulSet.SelectorLabels) == 0 {
				continue
			}
			for j := range space.Pods {
				pod := &space.Pods[j]
				if framework.LabelsMatchOnly(pod.LabelsMap, statefulSet.SelectorLabels) {
					edges = append(edges, framework.CreateEdge(statefulSet, pod, "BHK_ManagedBy"))
				}
			}
		}
	}
	return edges
}

type jobEdgesRule struct{}

func (r jobEdgesRule) Name() string { return "job" }

func (r jobEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Jobs {
			job := &space.Jobs[i]
			if len(job.SelectorLabels) == 0 {
				continue
			}
			for j := range space.Pods {
				pod := &space.Pods[j]
				if framework.LabelsMatchOnly(pod.LabelsMap, job.SelectorLabels) {
					edges = append(edges, framework.CreateEdge(job, pod, "BHK_ManagedBy"))
				}
			}
		}
	}
	return edges
}

type cronJobEdgesRule struct{}

func (r cronJobEdgesRule) Name() string { return "cronjob" }

func (r cronJobEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.CronJobs {
			cronJob := &space.CronJobs[i]
			if len(cronJob.SelectorLabels) == 0 {
				continue
			}
			for j := range space.Pods {
				pod := &space.Pods[j]
				if framework.LabelsMatchOnly(pod.LabelsMap, cronJob.SelectorLabels) {
					edges = append(edges, framework.CreateEdge(cronJob, pod, "BHK_ManagedBy"))
				}
			}
		}
	}
	return edges
}
