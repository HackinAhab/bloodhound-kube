package nodes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type PersistentVolumeClaim struct {
	GraphNodeBase
}

func BuildPVCNode(obj runtime.Object) (BuildResult, bool) {
	pvc, ok := obj.(*corev1.PersistentVolumeClaim)
	if !ok || pvc == nil {
		return BuildResult{}, false
	}
	name := pvc.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := pvc.Namespace
	labelsMap := StringMapToAnyMap(pvc.Labels)
	annotationsMap := StringMapToAnyMap(pvc.Annotations)

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: PersistentVolumeClaim{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("PersistentVolumeClaim", namespace, name),
				Kinds:          []string{"PersistentVolumeClaim"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("PersistentVolumeClaim", namespace, name),
			Kinds:      []string{"PersistentVolumeClaim"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
