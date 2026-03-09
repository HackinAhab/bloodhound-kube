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

	base := NewGraphNodeBase("PersistentVolumeClaim", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: PersistentVolumeClaim{
			GraphNodeBase: base,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}
