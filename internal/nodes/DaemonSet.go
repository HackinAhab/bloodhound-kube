package nodes

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	RegisterTyped(appsv1.SchemeGroupVersion.WithKind("DaemonSet"), BuildDaemonSetNode)
}

type DaemonSetCore struct {
	GraphNodeBase
	SelectorLabels map[string]string
	ServiceAccount string
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

	selectorMap := StringMapToAnyMap(set.Spec.Selector.MatchLabels)
	serviceAccount := set.Spec.Template.Spec.ServiceAccountName

	properties := map[string]any{
		"name":           name,
		"namespace":      namespace,
		"labels":         MapToSortedList(labelsMap),
		"annotations":    MapToSortedList(annotationsMap),
		"selector":       MapToSortedList(selectorMap),
		"serviceAccount": serviceAccount,
	}

	base := NewGraphNodeBase("DaemonSet", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: DaemonSetCore{
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
