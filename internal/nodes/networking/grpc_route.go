package networking

import (
	. "bloodhound-kube/internal/nodes/framework"

	"k8s.io/apimachinery/pkg/runtime"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)



type GRPCRoute struct {
	GraphNodeBase
	BackendRefs       []HTTPRouteBackendRef
	ParentGatewayRefs []ParentGatewayRef
}

func BuildGRPCRouteNode(obj runtime.Object) (BuildResult, bool) {
	switch typed := obj.(type) {
	case *gatewayv1.GRPCRoute:
		return buildGRPCRouteFromV1(typed)
	case *gatewayv1alpha2.GRPCRoute:
		converted := gatewayv1.GRPCRoute(*typed)
		return buildGRPCRouteFromV1(&converted)
	default:
		return BuildResult{}, false
	}
}

func buildGRPCRouteFromV1(route *gatewayv1.GRPCRoute) (BuildResult, bool) {
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

	backendRefs, backendRefKeys := extractGRPCRouteBackendRefs(route.Spec.Rules, namespace)
	parentRefs := summarizeHTTPRouteParents(route.Spec.ParentRefs, namespace)
	parentGatewayRefs := extractParentGatewayRefs(route.Spec.ParentRefs, namespace)
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

	base := NewGraphNodeBase("GRPCRoute", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: GRPCRoute{
			GraphNodeBase:     base,
			BackendRefs:       backendRefs,
			ParentGatewayRefs: parentGatewayRefs,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func extractGRPCRouteBackendRefs(rules []gatewayv1.GRPCRouteRule, defaultNamespace string) ([]HTTPRouteBackendRef, []string) {
	if len(rules) == 0 {
		return []HTTPRouteBackendRef{}, []string{}
	}
	keys := map[string]struct{}{}
	refs := map[string]HTTPRouteBackendRef{}
	for _, rule := range rules {
		for _, backend := range rule.BackendRefs {
			extractGatewayServiceBackendRef(backend.BackendRef, defaultNamespace, refs, keys)
		}
	}
	return backendRefsFromMap(refs, keys)
}
