package networking

import (
	. "bloodhound-kube/internal/nodes/framework"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)



type Service struct {
	GraphNodeBase
	ServiceType string
	ExternalIPs []string
	SelectorMap map[string]string
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
	externalIPs := extractServiceExternalIPs(svc)

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"selector":    MapToSortedList(selectorMap),
		"serviceType": serviceType,
		"clusterIP":   svc.Spec.ClusterIP,
		"externalIPs": externalIPs,
	}

	base := NewGraphNodeBase("Service", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Service{
			GraphNodeBase: base,
			ServiceType:   serviceType,
			ExternalIPs:   externalIPs,
			SelectorMap:   svc.Spec.Selector,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func extractServiceExternalIPs(svc *corev1.Service) []string {
	if svc == nil {
		return nil
	}

	seen := map[string]struct{}{}
	results := make([]string, 0, len(svc.Spec.ExternalIPs)+len(svc.Status.LoadBalancer.Ingress))

	add := func(value string) {
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		results = append(results, value)
	}

	for _, ip := range svc.Spec.ExternalIPs {
		add(ip)
	}

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		add(ingress.IP)
		add(ingress.Hostname)
	}

	sort.Strings(results)
	return results
}
