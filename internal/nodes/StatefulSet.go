package nodes

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	RegisterTyped(appsv1.SchemeGroupVersion.WithKind("StatefulSet"), BuildStatefulSetNode)
}

type StatefulSetCore struct {
	GraphNodeBase
	SelectorLabels map[string]string
	ServiceAccount string
}

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

	selectorMap := StringMapToAnyMap(set.Spec.Selector.MatchLabels)
	serviceName := set.Spec.ServiceName
	serviceAccount := set.Spec.Template.Spec.ServiceAccountName

	replicas := 0
	if set.Spec.Replicas != nil {
		replicas = int(*set.Spec.Replicas)
	}

	properties := map[string]any{
		"name":           name,
		"namespace":      namespace,
		"labels":         MapToSortedList(labelsMap),
		"annotations":    MapToSortedList(annotationsMap),
		"replicas":       replicas,
		"selector":       MapToSortedList(selectorMap),
		"serviceAccount": serviceAccount,
		"serviceName":    serviceName,
	}

	base := NewGraphNodeBase("StatefulSet", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: StatefulSetCore{
			GraphNodeBase:  base,
			SelectorLabels: set.Spec.Selector.MatchLabels,
			ServiceAccount: serviceAccount,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}
