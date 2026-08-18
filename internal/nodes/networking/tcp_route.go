package networking

import (
	. "bloodhound-kube/internal/nodes/framework"

	"k8s.io/apimachinery/pkg/runtime"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)



type TCPRoute struct {
	GraphNodeBase
	BackendRefs       []HTTPRouteBackendRef
	ParentGatewayRefs []ParentGatewayRef
}

func (r *TCPRoute) GetParentGatewayRefs() []ParentGatewayRef { return r.ParentGatewayRefs }
func (r *TCPRoute) GetBackendRefs() []HTTPRouteBackendRef     { return r.BackendRefs }

func BuildTCPRouteNode(obj runtime.Object) (BuildResult, bool) {
	route, ok := obj.(*gatewayv1alpha2.TCPRoute)
	if !ok || route == nil {
		return BuildResult{}, false
	}

	name := route.Name
	if name == "" {
		return BuildResult{}, false
	}
	namespace := route.Namespace
	labelsMap := StringMapToAnyMap(route.Labels)
	annotationsMap := StringMapToAnyMap(route.Annotations)

	backendRefs, backendRefKeys := extractTCPRouteBackendRefs(route.Spec.Rules, namespace)
	parentRefs := summarizeHTTPRouteParents(route.Spec.ParentRefs, namespace)
	parentGatewayRefs := extractParentGatewayRefs(route.Spec.ParentRefs, namespace)

	properties := map[string]any{
		"name":           name,
		"namespace":      namespace,
		"labels":         MapToSortedList(labelsMap),
		"annotations":    MapToSortedList(annotationsMap),
		"parentRefs":     parentRefs,
		"backendRefKeys": backendRefKeys,
	}

	base := NewGraphNodeBase("BHK_TCPRoute", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: TCPRoute{
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

func extractTCPRouteBackendRefs(rules []gatewayv1alpha2.TCPRouteRule, defaultNamespace string) ([]HTTPRouteBackendRef, []string) {
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

func extractGatewayServiceBackendRef(backend gatewayv1.BackendRef, defaultNamespace string, refs map[string]HTTPRouteBackendRef, keys map[string]struct{}) {
	kind := ""
	if backend.Kind != nil {
		kind = string(*backend.Kind)
	}
	if kind != "" && kind != "Service" {
		return
	}
	name := string(backend.Name)
	if name == "" {
		return
	}
	ns := defaultNamespace
	if backend.Namespace != nil {
		ns = string(*backend.Namespace)
	}
	key := ns + "/" + name
	keys[key] = struct{}{}
	refs[key] = HTTPRouteBackendRef{
		Namespace: ns,
		Name:      name,
	}
}
