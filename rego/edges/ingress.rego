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

# Assumes all routes to an Ingress are externally accessible.  
# External routes to Ingress
external_routes_to_ingress contains edge if {
	external := input.cluster_scoped.external[_]
	ingress_ns := input.namespaces[ns]
	ingress := ingress_ns.ingress[_]

	edge := helpers.create_edge(external, ingress, "ExternalRoutesTo")
}