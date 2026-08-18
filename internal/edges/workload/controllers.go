package workload

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/workload"
)

type workloadController[P any] interface {
	*P
	nodefw.EdgeNode
	GetSelectorLabels() map[string]string
}

func applyControllerRule[T any, P workloadController[T]](ctx *framework.Context, getItems func(*model.Namespace) []T) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range getItems(space) {
			controller := P(&getItems(space)[i])
			if len(controller.GetSelectorLabels()) == 0 {
				continue
			}
			for j := range space.Pods {
				pod := &space.Pods[j]
				if framework.LabelsMatchOnly(pod.LabelsMap, controller.GetSelectorLabels()) {
					edges = append(edges, framework.CreateEdge(controller, pod, "BHK_ManagedBy"))
				}
			}
		}
	}
	return edges
}

type deploymentEdgesRule struct{}

func (r deploymentEdgesRule) Name() string { return "deployment" }
func (r deploymentEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	return applyControllerRule[workload.Deployment, *workload.Deployment](ctx, func(s *model.Namespace) []workload.Deployment { return s.Deployments })
}

type daemonSetEdgesRule struct{}

func (r daemonSetEdgesRule) Name() string { return "daemonset" }
func (r daemonSetEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	return applyControllerRule[workload.DaemonSetCore, *workload.DaemonSetCore](ctx, func(s *model.Namespace) []workload.DaemonSetCore { return s.DaemonSets })
}

type statefulSetEdgesRule struct{}

func (r statefulSetEdgesRule) Name() string { return "statefulset" }
func (r statefulSetEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	return applyControllerRule[workload.StatefulSetCore, *workload.StatefulSetCore](ctx, func(s *model.Namespace) []workload.StatefulSetCore { return s.StatefulSets })
}

type jobEdgesRule struct{}

func (r jobEdgesRule) Name() string { return "job" }
func (r jobEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	return applyControllerRule[workload.Job, *workload.Job](ctx, func(s *model.Namespace) []workload.Job { return s.Jobs })
}

type cronJobEdgesRule struct{}

func (r cronJobEdgesRule) Name() string { return "cronjob" }
func (r cronJobEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	return applyControllerRule[workload.CronJob, *workload.CronJob](ctx, func(s *model.Namespace) []workload.CronJob { return s.CronJobs })
}
