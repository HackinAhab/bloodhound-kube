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
	entries := MapEntriesSorted(data)

	properties := map[string]any{
		"name":          name,
		"namespace":     namespace,
		"labels":        MapToSortedList(labelsMap),
		"annotations":   MapToSortedList(annotationsMap),
		"dataKeys":      keys,
		"dataKeysCount": len(keys),
		"dataEntries":   entries,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: ConfigMap{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("ConfigMap", namespace, name),
				Kinds:          []string{"ConfigMap"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			Data: data,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("ConfigMap", namespace, name),
			Kinds:      []string{"ConfigMap"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}
