package nodes

import (
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	RegisterTyped(batchv1.SchemeGroupVersion.WithKind("CronJob"), BuildCronJobNode)
}

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
		selectorLabels = cronJob.Spec.JobTemplate.Spec.Selector.MatchLabels
	}
	selectorMap := StringMapToAnyMap(selectorLabels)
	serviceAccount := cronJob.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName

	suspend := false
	if cronJob.Spec.Suspend != nil {
		suspend = *cronJob.Spec.Suspend
	}

	properties := map[string]any{
		"name":              name,
		"namespace":         namespace,
		"labels":            MapToSortedList(labelsMap),
		"annotations":       MapToSortedList(annotationsMap),
		"selector":          MapToSortedList(selectorMap),
		"serviceAccount":    serviceAccount,
		"schedule":          cronJob.Spec.Schedule,
		"suspend":           suspend,
		"concurrencyPolicy": string(cronJob.Spec.ConcurrencyPolicy),
	}

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
