package nodes

import (
	"k8s.io/apimachinery/pkg/runtime"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type HTTPRoute struct {
	GraphNodeBase
	BackendRefs       []HTTPRouteBackendRef
	ParentGatewayRefs []ParentGatewayRef
}

type HTTPRouteBackendRef struct {
	Namespace string
	Name      string
}

func BuildHTTPRouteNode(obj runtime.Object) (BuildResult, bool) {
	switch typed := obj.(type) {
	case *gatewayv1.HTTPRoute:
		return buildHTTPRouteFromV1(typed)
	case *gatewayv1beta1.HTTPRoute:
		converted := gatewayv1.HTTPRoute(*typed)
		return buildHTTPRouteFromV1(&converted)
	default:
		return BuildResult{}, false
	}
}

func buildHTTPRouteFromV1(route *gatewayv1.HTTPRoute) (BuildResult, bool) {
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

	backendRefs, backendRefKeys := extractHTTPRouteBackendRefs(route.Spec.Rules, namespace)
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

	base := NewGraphNodeBase("HTTPRoute", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: HTTPRoute{
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

func extractHTTPRouteBackendRefs(rules []gatewayv1.HTTPRouteRule, defaultNamespace string) ([]HTTPRouteBackendRef, []string) {
	if len(rules) == 0 {
		return []HTTPRouteBackendRef{}, []string{}
	}
	keys := map[string]struct{}{}
	refs := map[string]HTTPRouteBackendRef{}
	for _, rule := range rules {
		for _, backend := range rule.BackendRefs {
			kind := ""
			if backend.Kind != nil {
				kind = string(*backend.Kind)
			}
			if kind != "" && kind != "Service" {
				continue
			}
			name := string(backend.Name)
			if name == "" {
				continue
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
	}
	return backendRefsFromMap(refs, keys)
}

func backendRefsFromMap(refs map[string]HTTPRouteBackendRef, keys map[string]struct{}) ([]HTTPRouteBackendRef, []string) {
	if len(keys) == 0 {
		return []HTTPRouteBackendRef{}, []string{}
	}
	sortedKeys := SortedSetKeys(keys)
	items := make([]HTTPRouteBackendRef, 0, len(sortedKeys))
	for _, key := range sortedKeys {
		if ref, ok := refs[key]; ok {
			items = append(items, ref)
		}
	}
	return items, sortedKeys
}

func summarizeHTTPRouteParents(parents []gatewayv1.ParentReference, defaultNamespace string) []string {
	if len(parents) == 0 {
		return []string{}
	}
	entries := make([]string, 0, len(parents))
	for _, parent := range parents {
		name := string(parent.Name)
		if name == "" {
			continue
		}
		kind := "Gateway"
		if parent.Kind != nil && *parent.Kind != "" {
			kind = string(*parent.Kind)
		}
		ns := defaultNamespace
		if parent.Namespace != nil && *parent.Namespace != "" {
			ns = string(*parent.Namespace)
		}
		if ns != "" {
			entries = append(entries, kind+":"+ns+"/"+name)
			continue
		}
		entries = append(entries, kind+":"+name)
	}
	return entries
}

func extractParentGatewayRefs(parents []gatewayv1.ParentReference, defaultNamespace string) []ParentGatewayRef {
	if len(parents) == 0 {
		return []ParentGatewayRef{}
	}
	keys := map[string]struct{}{}
	items := map[string]ParentGatewayRef{}
	for _, parent := range parents {
		if !isGatewayParentRef(parent) {
			continue
		}
		name := string(parent.Name)
		if name == "" {
			continue
		}
		ns := defaultNamespace
		if parent.Namespace != nil && *parent.Namespace != "" {
			ns = string(*parent.Namespace)
		}
		key := ns + "/" + name
		keys[key] = struct{}{}
		items[key] = ParentGatewayRef{Namespace: ns, Name: name}
	}
	if len(keys) == 0 {
		return []ParentGatewayRef{}
	}
	sortedKeys := SortedSetKeys(keys)
	result := make([]ParentGatewayRef, 0, len(sortedKeys))
	for _, key := range sortedKeys {
		if ref, ok := items[key]; ok {
			result = append(result, ref)
		}
	}
	return result
}

func isGatewayParentRef(parent gatewayv1.ParentReference) bool {
	if parent.Kind != nil && *parent.Kind != "" && string(*parent.Kind) != "Gateway" {
		return false
	}
	if parent.Group != nil && *parent.Group != "" && string(*parent.Group) != gatewayv1.GroupName {
		return false
	}
	return true
}

func httpRouteHostnames(hostnames []gatewayv1.Hostname) []string {
	if len(hostnames) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(hostnames))
	for _, hostname := range hostnames {
		items = append(items, string(hostname))
	}
	return items
}
