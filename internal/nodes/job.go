package nodes

import (
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	RegisterTyped(batchv1.SchemeGroupVersion.WithKind("Job"), BuildJobNode)
}

type Job struct {
	GraphNodeBase
	SelectorLabels map[string]string
	ServiceAccount string
}

func BuildJobNode(obj runtime.Object) (BuildResult, bool) {
	job, ok := obj.(*batchv1.Job)
	if !ok || job == nil {
		return BuildResult{}, false
	}
	name := job.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := job.Namespace
	labelsMap := StringMapToAnyMap(job.Labels)
	annotationsMap := StringMapToAnyMap(job.Annotations)

	selectorLabels := map[string]string{}
	if job.Spec.Selector != nil {
		selectorLabels = job.Spec.Selector.MatchLabels
	}
	selectorMap := StringMapToAnyMap(selectorLabels)
	serviceAccount := job.Spec.Template.Spec.ServiceAccountName

	parallelism := 0
	if job.Spec.Parallelism != nil {
		parallelism = int(*job.Spec.Parallelism)
	}
	completions := 0
	if job.Spec.Completions != nil {
		completions = int(*job.Spec.Completions)
	}

	properties := map[string]any{
		"name":           name,
		"namespace":      namespace,
		"labels":         MapToSortedList(labelsMap),
		"annotations":    MapToSortedList(annotationsMap),
		"selector":       MapToSortedList(selectorMap),
		"serviceAccount": serviceAccount,
		"parallelism":    parallelism,
		"completions":    completions,
	}

	base := NewGraphNodeBase("Job", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Job{
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
