package networking

import (
	. "bloodhound-kube/internal/nodes/framework"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
)



type Ingress struct {
	GraphNodeBase
	BackendRefs []HTTPRouteBackendRef
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

	backendRefs, backendRefKeys := extractIngressBackendRefs(ingress.Spec, namespace)
	ingressURLs := extractIngressURLs(ingress.Spec)
	properties := map[string]any{
		"name":           name,
		"namespace":      namespace,
		"labels":         MapToSortedList(labelsMap),
		"annotations":    MapToSortedList(annotationsMap),
		"urls":           ingressURLs,
		"backendRefKeys": backendRefKeys,
	}

	base := NewGraphNodeBase("BHK_Ingress", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Ingress{
			GraphNodeBase: base,
			BackendRefs:   backendRefs,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func extractIngressBackendRefs(spec networkingv1.IngressSpec, defaultNamespace string) ([]HTTPRouteBackendRef, []string) {
	keys := map[string]struct{}{}
	refs := map[string]HTTPRouteBackendRef{}

	if spec.DefaultBackend != nil && spec.DefaultBackend.Service != nil {
		name := spec.DefaultBackend.Service.Name
		if name != "" {
			key := defaultNamespace + "/" + name
			keys[key] = struct{}{}
			refs[key] = HTTPRouteBackendRef{Namespace: defaultNamespace, Name: name}
		}
	}

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
				key := defaultNamespace + "/" + name
				keys[key] = struct{}{}
				refs[key] = HTTPRouteBackendRef{Namespace: defaultNamespace, Name: name}
			}
		}
	}

	return backendRefsFromMap(refs, keys)
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
