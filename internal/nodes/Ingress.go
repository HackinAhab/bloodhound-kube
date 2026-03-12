package nodes

import (
	"strings"

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
	ingressURLs := extractIngressURLs(ingress.Spec)
	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"urls":        ingressURLs,
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

func extractIngressURLs(spec networkingv1.IngressSpec) []string {
	urls := map[string]struct{}{}
	tlsHosts := map[string]struct{}{}

	for _, tls := range spec.TLS {
		for _, host := range tls.Hosts {
			host = strings.TrimSpace(host)
			if host != "" {
				tlsHosts[host] = struct{}{}
			}
		}
	}

	for _, rule := range spec.Rules {
		host := strings.TrimSpace(rule.Host)
		if host == "" {
			continue
		}

		scheme := "http"
		if _, isTLS := tlsHosts[host]; isTLS {
			scheme = "https"
		}

		if rule.HTTP == nil || len(rule.HTTP.Paths) == 0 {
			urls[scheme+"://"+host] = struct{}{}
			continue
		}

		for _, ingressPath := range rule.HTTP.Paths {
			path := ingressPath.Path
			if path == "" {
				path = "/"
			}
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			urls[scheme+"://"+host+path] = struct{}{}
		}
	}

	return SortedSetKeys(urls)
}
