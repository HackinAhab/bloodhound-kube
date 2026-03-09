package nodes

import (
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Ingress struct {
	GraphNodeBase
	BackendServices []string
}

func BuildIngressNode(obj runtime.Object) (BuildResult, bool) {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok || ingress == nil {
		return BuildResult{}, false
	}
	name := ingress.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := ingress.Namespace
	labelsMap := StringMapToAnyMap(ingress.Labels)
	annotationsMap := StringMapToAnyMap(ingress.Annotations)

	backendServices := extractIngressBackendServices(ingress.Spec)
	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	base := NewGraphNodeBase("Ingress", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Ingress{
			GraphNodeBase:   base,
			BackendServices: backendServices,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func extractIngressBackendServices(spec networkingv1.IngressSpec) []string {
	services := map[string]struct{}{}
	for _, rule := range spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}
			name := path.Backend.Service.Name
			if name != "" {
				services[name] = struct{}{}
			}
		}
	}
	return SortedSetKeys(services)
}
