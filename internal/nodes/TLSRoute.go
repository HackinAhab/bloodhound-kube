package nodes

import (
	"k8s.io/apimachinery/pkg/runtime"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type TLSRoute struct {
	GraphNodeBase
	BackendRefs []HTTPRouteBackendRef
}

func BuildTLSRouteNode(obj runtime.Object) (BuildResult, bool) {
	switch typed := obj.(type) {
	case *gatewayv1.TLSRoute:
		return buildTLSRouteFromV1(typed)
	case *gatewayv1alpha2.TLSRoute:
		return buildTLSRouteFromV1Alpha2(typed)
	default:
		return BuildResult{}, false
	}
}

func buildTLSRouteFromV1(route *gatewayv1.TLSRoute) (BuildResult, bool) {
	if route == nil {
		return BuildResult{}, false
	}
	name := route.Name
	if name == "" {
		return BuildResult{}, false
	}
	namespace := route.Namespace
	labelsMap := StringMapToAnyMap(route.Labels)
	annotationsMap := StringMapToAnyMap(route.Annotations)

	backendRefs, backendRefKeys := extractTLSRouteBackendRefs(route.Spec.Rules, namespace)
	parentRefs := summarizeHTTPRouteParents(route.Spec.ParentRefs, namespace)
	hostnames := httpRouteHostnames(route.Spec.Hostnames)

	properties := map[string]any{
		"name":           name,
		"namespace":      namespace,
		"labels":         MapToSortedList(labelsMap),
		"annotations":    MapToSortedList(annotationsMap),
		"hostnames":      hostnames,
		"parentRefs":     parentRefs,
		"backendRefKeys": backendRefKeys,
	}

	base := NewGraphNodeBase("TLSRoute", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: TLSRoute{
			GraphNodeBase: base,
			BackendRefs:   backendRefs,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func buildTLSRouteFromV1Alpha2(route *gatewayv1alpha2.TLSRoute) (BuildResult, bool) {
	if route == nil {
		return BuildResult{}, false
	}
	name := route.Name
	if name == "" {
		return BuildResult{}, false
	}
	namespace := route.Namespace
	labelsMap := StringMapToAnyMap(route.Labels)
	annotationsMap := StringMapToAnyMap(route.Annotations)

	backendRefs, backendRefKeys := extractTLSRouteBackendRefsV1Alpha2(route.Spec.Rules, namespace)
	parentRefs := summarizeHTTPRouteParents(route.Spec.ParentRefs, namespace)
	hostnames := httpRouteHostnames(route.Spec.Hostnames)

	properties := map[string]any{
		"name":           name,
		"namespace":      namespace,
		"labels":         MapToSortedList(labelsMap),
		"annotations":    MapToSortedList(annotationsMap),
		"hostnames":      hostnames,
		"parentRefs":     parentRefs,
		"backendRefKeys": backendRefKeys,
	}

	base := NewGraphNodeBase("TLSRoute", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: TLSRoute{
			GraphNodeBase: base,
			BackendRefs:   backendRefs,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func extractTLSRouteBackendRefs(rules []gatewayv1.TLSRouteRule, defaultNamespace string) ([]HTTPRouteBackendRef, []string) {
	if len(rules) == 0 {
		return []HTTPRouteBackendRef{}, []string{}
	}
	keys := map[string]struct{}{}
	refs := map[string]HTTPRouteBackendRef{}
	for _, rule := range rules {
		for _, backend := range rule.BackendRefs {
			extractGatewayServiceBackendRef(backend, defaultNamespace, refs, keys)
		}
	}
	return backendRefsFromMap(refs, keys)
}

func extractTLSRouteBackendRefsV1Alpha2(rules []gatewayv1alpha2.TLSRouteRule, defaultNamespace string) ([]HTTPRouteBackendRef, []string) {
	if len(rules) == 0 {
		return []HTTPRouteBackendRef{}, []string{}
	}
	keys := map[string]struct{}{}
	refs := map[string]HTTPRouteBackendRef{}
	for _, rule := range rules {
		for _, backend := range rule.BackendRefs {
			extractGatewayServiceBackendRef(gatewayv1.BackendRef(backend), defaultNamespace, refs, keys)
		}
	}
	return backendRefsFromMap(refs, keys)
}
