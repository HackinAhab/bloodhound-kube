package kubernetes.relationships.workloads

import rego.v1
import data.kubernetes.helpers


# Ingress routes to Service
ingress_routes_to_service contains edge if {
	namespace := input.namespaces[ns]
	ingress := namespace.ingress[_]
	service := namespace.service[_]
	
	# Check if any ingress rule references this service
	rule := ingress.properties.rules[_]
	path := rule.paths[_]
	path.backendService == service.properties.name
	
	edge := helpers.create_edge(ingress, service, "RoutesTo")
}
