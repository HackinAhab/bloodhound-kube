package nodes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Service struct {
	GraphNodeBase
}

func BuildServiceNode(obj runtime.Object) (BuildResult, bool) {
	svc, ok := obj.(*corev1.Service)
	if !ok || svc == nil {
		return BuildResult{}, false
	}
	name := svc.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := svc.Namespace
	labelsMap := StringMapToAnyMap(svc.Labels)
	annotationsMap := StringMapToAnyMap(svc.Annotations)
	selectorMap := StringMapToAnyMap(svc.Spec.Selector)

	serviceType := string(svc.Spec.Type)

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"selector":    MapToSortedList(selectorMap),
		"serviceType": serviceType,
		"clusterIP":   svc.Spec.ClusterIP,
	}

	base := NewGraphNodeBase("Service", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Service{
			GraphNodeBase: base,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}
