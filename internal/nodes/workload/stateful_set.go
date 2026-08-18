package workload

import (
	. "bloodhound-kube/internal/nodes/framework"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type StatefulSetCore struct {
	GraphNodeBase
	SelectorLabels map[string]string
	ServiceAccount string
	EnvDefinitions []EnvDefinition
}

func (s *StatefulSetCore) GetSelectorLabels() map[string]string { return s.SelectorLabels }

func BuildStatefulSetNode(obj runtime.Object) (BuildResult, bool) {
	set, ok := obj.(*appsv1.StatefulSet)
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
	serviceName := set.Spec.ServiceName
	serviceAccount := set.Spec.Template.Spec.ServiceAccountName
	envDefinitions := buildEnvDefinitionsFromContainers(set.Spec.Template.Spec.Containers, false, "StatefulSet", name)
	envDefinitions = append(envDefinitions, buildEnvDefinitionsFromContainers(set.Spec.Template.Spec.InitContainers, true, "StatefulSet", name)...)

	replicas := I32(set.Spec.Replicas)

	properties := Props(name, namespace, labelsMap, annotationsMap)
	properties["replicas"] = replicas
	properties["selector"] = MapToSortedList(selectorMap)
	properties["serviceAccount"] = serviceAccount
	properties["serviceName"] = serviceName
	properties["envDefinitions"] = envDefinitions

	base := NewGraphNodeBase("BHK_StatefulSet", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: StatefulSetCore{
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
