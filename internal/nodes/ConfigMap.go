package nodes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ConfigMap struct {
	GraphNodeBase
	Data map[string]any
}

func BuildConfigMapNode(obj runtime.Object) (BuildResult, bool) {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok || cm == nil {
		return BuildResult{}, false
	}
	name := cm.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := cm.Namespace
	labelsMap := StringMapToAnyMap(cm.Labels)
	annotationsMap := StringMapToAnyMap(cm.Annotations)

	data := StringMapToAnyMap(cm.Data)
	keys := MapKeysSorted(data)
	entries := MapToSortedList(data)

	properties := map[string]any{
		"name":          name,
		"namespace":     namespace,
		"labels":        MapToSortedList(labelsMap),
		"annotations":   MapToSortedList(annotationsMap),
		"dataKeys":      keys,
		"dataKeysCount": len(keys),
		"dataEntries":   entries,
	}

	base := NewGraphNodeBase("ConfigMap", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: ConfigMap{
			GraphNodeBase: base,
			Data:          data,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}
