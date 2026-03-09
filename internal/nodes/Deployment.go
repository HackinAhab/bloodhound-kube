package nodes

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Deployment struct {
	GraphNodeBase
	SelectorLabels map[string]string
	ServiceAccount string
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

	selectorLabels := deploy.Spec.Selector.MatchLabels
	selectorMap := StringMapToAnyMap(selectorLabels)
	serviceAccount := deploy.Spec.Template.Spec.ServiceAccountName

	replicas := 0
	if deploy.Spec.Replicas != nil {
		replicas = int(*deploy.Spec.Replicas)
	}

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"replicas":    replicas,
		"selector":    MapToSortedList(selectorMap),
	}

	base := NewGraphNodeBase("Deployment", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Deployment{
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
