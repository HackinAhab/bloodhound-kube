package nodes

import (
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Ingress struct {
	GraphNodeBase
	BackendServices []string
	TLS             []any
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
	tlsEntries := ingressTLSToAnySlice(ingress.Spec.TLS)

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Ingress{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("Ingress", namespace, name),
				Kinds:          []string{"Ingress"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			BackendServices: backendServices,
			TLS:             tlsEntries,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Ingress", namespace, name),
			Kinds:      []string{"Ingress"},
			Properties: properties,
		},
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
	return setToSortedList(services)
}

func ingressTLSToAnySlice(tls []networkingv1.IngressTLS) []any {
	if len(tls) == 0 {
		return []any{}
	}
	items := make([]any, 0, len(tls))
	for _, entry := range tls {
		items = append(items, map[string]any{
			"hosts":      entry.Hosts,
			"secretName": entry.SecretName,
		})
	}
	return items
}
