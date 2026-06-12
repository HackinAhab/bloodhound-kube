package workload

import (
	. "bloodhound-kube/internal/nodes/framework"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Deployment struct {
	GraphNodeBase
	SelectorLabels map[string]string
	ServiceAccount string
	EnvDefinitions []EnvDefinition
}

func BuildDeploymentNode(obj runtime.Object) (BuildResult, bool) {
	deploy, ok := obj.(*appsv1.Deployment)
	if !ok || deploy == nil {
		return BuildResult{}, false
	}
	name := deploy.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := deploy.Namespace
	labelsMap := StringMapToAnyMap(deploy.Labels)
	annotationsMap := StringMapToAnyMap(deploy.Annotations)

	selectorLabels := Labels(deploy.Spec.Selector.MatchLabels)
	selectorMap := StringMapToAnyMap(selectorLabels)
	serviceAccount := deploy.Spec.Template.Spec.ServiceAccountName
	envDefinitions := buildEnvDefinitionsFromContainers(deploy.Spec.Template.Spec.Containers, false, "Deployment", name)
	envDefinitions = append(envDefinitions, buildEnvDefinitionsFromContainers(deploy.Spec.Template.Spec.InitContainers, true, "Deployment", name)...)

	replicas := I32(deploy.Spec.Replicas)

	properties := Props(name, namespace, labelsMap, annotationsMap)
	properties["replicas"] = replicas
	properties["selector"] = MapToSortedList(selectorMap)
	properties["envDefinitions"] = envDefinitions

	base := NewGraphNodeBase("Deployment", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Deployment{
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
