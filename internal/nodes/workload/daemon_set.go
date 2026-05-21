package workload

import (
	. "bloodhound-kube/internal/nodes/framework"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DaemonSetCore struct {
	GraphNodeBase
	SelectorLabels map[string]string
	ServiceAccount string
	EnvDefinitions []EnvDefinition
}

func BuildDaemonSetNode(obj runtime.Object) (BuildResult, bool) {
	set, ok := obj.(*appsv1.DaemonSet)
	if !ok || set == nil {
		return BuildResult{}, false
	}
	name := set.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := set.Namespace
	labelsMap := StringMapToAnyMap(set.Labels)
	annotationsMap := StringMapToAnyMap(set.Annotations)

	selectorLabels := Labels(set.Spec.Selector.MatchLabels)
	selectorMap := StringMapToAnyMap(selectorLabels)
	serviceAccount := set.Spec.Template.Spec.ServiceAccountName
	envDefinitions := buildEnvDefinitionsFromContainers(set.Spec.Template.Spec.Containers, false, "DaemonSet", name)
	envDefinitions = append(envDefinitions, buildEnvDefinitionsFromContainers(set.Spec.Template.Spec.InitContainers, true, "DaemonSet", name)...)

	properties := Props(name, namespace, labelsMap, annotationsMap)
	properties["selector"] = MapToSortedList(selectorMap)
	properties["serviceAccount"] = serviceAccount
	properties["envDefinitions"] = envDefinitions

	base := NewGraphNodeBase("DaemonSet", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: DaemonSetCore{
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
