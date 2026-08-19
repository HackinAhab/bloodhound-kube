package route

import (
	. "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/networking"

	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Route (OpenShift route.openshift.io) is not build-tag-gated — it's a
// deliberate, always-on integration, mirroring the addons/scc package.

func Register() {
	RegisterTyped(routev1.SchemeGroupVersion.WithKind("Route"), BuildRouteNode)
}

type Route struct {
	GraphNodeBase
	BackendRefs []networking.HTTPRouteBackendRef
}

func (r *Route) GetBackendRefs() []networking.HTTPRouteBackendRef { return r.BackendRefs }

func BuildRouteNode(obj runtime.Object) (BuildResult, bool) {
	route, ok := obj.(*routev1.Route)
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

	backendRefs, backendRefKeys := extractRouteBackendRefs(route.Spec, namespace)
	url := extractRouteURL(route.Spec)

	properties := map[string]any{
		"name":           name,
		"namespace":      namespace,
		"labels":         MapToSortedList(labelsMap),
		"annotations":    MapToSortedList(annotationsMap),
		"backendRefKeys": backendRefKeys,
	}
	if url != "" {
		properties["urls"] = []string{url}
	} else {
		properties["urls"] = []string{}
	}

	base := NewGraphNodeBase("BHK_Route", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Route{
			GraphNodeBase: base,
			BackendRefs:   backendRefs,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func extractRouteBackendRefs(spec routev1.RouteSpec, defaultNamespace string) ([]networking.HTTPRouteBackendRef, []string) {
	keys := map[string]struct{}{}
	refs := map[string]networking.HTTPRouteBackendRef{}

	addTarget := func(target routev1.RouteTargetReference) {
		name := target.Name
		if name == "" || (target.Kind != "" && target.Kind != "Service") {
			return
		}
		key := defaultNamespace + "/" + name
		keys[key] = struct{}{}
		refs[key] = networking.HTTPRouteBackendRef{Namespace: defaultNamespace, Name: name}
	}

	addTarget(spec.To)
	for _, alt := range spec.AlternateBackends {
		addTarget(alt)
	}

	if len(keys) == 0 {
		return []networking.HTTPRouteBackendRef{}, []string{}
	}
	sortedKeys := SortedSetKeys(keys)
	items := make([]networking.HTTPRouteBackendRef, 0, len(sortedKeys))
	for _, key := range sortedKeys {
		items = append(items, refs[key])
	}
	return items, sortedKeys
}

func extractRouteURL(spec routev1.RouteSpec) string {
	host := spec.Host
	if host == "" {
		return ""
	}
	scheme := "http"
	if spec.TLS != nil {
		scheme = "https"
	}
	path := spec.Path
	if path == "" {
		path = "/"
	}
	return scheme + "://" + host + path
}
