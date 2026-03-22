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

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"selector":    MapToSortedList(selectorMap),
		"serviceName": serviceName,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: StatefulSetCore{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("StatefulSet", namespace, name),
				Kinds:          []string{"StatefulSet"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("StatefulSet", namespace, name),
			Kinds:      []string{"StatefulSet"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
