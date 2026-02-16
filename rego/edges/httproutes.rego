package kubernetes.relationships.httproutes

import rego.v1

import data.kubernetes.helpers

# HTTPRoute routes to Service via backendRefs
httproute_routes_to_service contains edge if {
	route_ns := input.namespaces[ns]
	route := route_ns.httproute[_]

	key := object.get(route.properties, "backendRefKeys", object.get(route.properties, "backend_ref_keys", []))[_]
	parts := split(key, "/")
	backend_ns := parts[0]
	backend_name := parts[1]
	backend_ns != ""

	svc_ns := input.namespaces[backend_ns]
	service := svc_ns.service[_]
	service.properties.name == backend_name

	edge := helpers.create_edge(route, service, "RoutesTo")
}


httproute_routes_to_service contains edge if {
	route_ns := input.namespaces[ns]
	route := route_ns.httproute[_]

	key := object.get(route.properties, "backendRefKeys", object.get(route.properties, "backend_ref_keys", []))[_]
	parts := split(key, "/")
	backend_ns := parts[0]
	backend_name := parts[1]
	backend_ns == ""

	svc_ns := input.namespaces[route.properties.namespace]
	service := svc_ns.service[_]
	service.properties.name == backend_name

	edge := helpers.create_edge(route, service, "RoutesTo")
}

external_routes_to_httproute contains edge if {
	external := input.cluster_scoped.external[_]
	route_ns := input.namespaces[ns]
	route := route_ns.httproute[_]

	edge := helpers.create_edge(external, route, "ExternalRoutesTo")
}
