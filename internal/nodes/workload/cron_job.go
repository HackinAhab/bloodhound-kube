package workload

import (
	. "bloodhound-kube/internal/nodes/framework"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type CronJob struct {
	GraphNodeBase
	SelectorLabels map[string]string
	ServiceAccount string
}

func BuildCronJobNode(obj runtime.Object) (BuildResult, bool) {
	cronJob, ok := obj.(*batchv1.CronJob)
	if !ok || cronJob == nil {
		return BuildResult{}, false
	}
	name := cronJob.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := cronJob.Namespace
	labelsMap := StringMapToAnyMap(cronJob.Labels)
	annotationsMap := StringMapToAnyMap(cronJob.Annotations)

	selectorLabels := map[string]string{}
	if cronJob.Spec.JobTemplate.Spec.Selector != nil {
		selectorLabels = Labels(cronJob.Spec.JobTemplate.Spec.Selector.MatchLabels)
	}
	selectorMap := StringMapToAnyMap(selectorLabels)
	serviceAccount := cronJob.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName

	suspend := B(cronJob.Spec.Suspend)

	properties := Props(name, namespace, labelsMap, annotationsMap)
	properties["selector"] = MapToSortedList(selectorMap)
	properties["serviceAccount"] = serviceAccount
	properties["schedule"] = cronJob.Spec.Schedule
	properties["suspend"] = suspend
	properties["concurrencyPolicy"] = string(cronJob.Spec.ConcurrencyPolicy)

	base := NewGraphNodeBase("CronJob", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: CronJob{
			GraphNodeBase:  base,
			SelectorLabels: selectorLabels,
			ServiceAccount: serviceAccount,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}
