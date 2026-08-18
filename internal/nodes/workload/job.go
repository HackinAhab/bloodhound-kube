package workload

import (
	. "bloodhound-kube/internal/nodes/framework"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Job struct {
	GraphNodeBase
	SelectorLabels map[string]string
	ServiceAccount string
	EnvDefinitions []EnvDefinition
}

func (j *Job) GetSelectorLabels() map[string]string { return j.SelectorLabels }

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
		selectorLabels = Labels(job.Spec.Selector.MatchLabels)
	}
	selectorMap := StringMapToAnyMap(selectorLabels)
	serviceAccount := job.Spec.Template.Spec.ServiceAccountName
	envDefinitions := buildEnvDefinitionsFromContainers(job.Spec.Template.Spec.Containers, false, "Job", name)
	envDefinitions = append(envDefinitions, buildEnvDefinitionsFromContainers(job.Spec.Template.Spec.InitContainers, true, "Job", name)...)

	parallelism := I32(job.Spec.Parallelism)
	completions := I32(job.Spec.Completions)

	properties := Props(name, namespace, labelsMap, annotationsMap)
	properties["selector"] = MapToSortedList(selectorMap)
	properties["serviceAccount"] = serviceAccount
	properties["parallelism"] = parallelism
	properties["completions"] = completions
	properties["envDefinitions"] = envDefinitions

	base := NewGraphNodeBase("BHK_Job", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Job{
			GraphNodeBase:  base,
			SelectorLabels: selectorLabels,
			ServiceAccount: serviceAccount,
			EnvDefinitions: envDefinitions,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}
