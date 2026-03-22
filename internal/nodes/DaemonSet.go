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

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"selector":    MapToSortedList(selectorMap),
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: DaemonSetCore{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("DaemonSet", namespace, name),
				Kinds:          []string{"DaemonSet"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("DaemonSet", namespace, name),
			Kinds:      []string{"DaemonSet"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
