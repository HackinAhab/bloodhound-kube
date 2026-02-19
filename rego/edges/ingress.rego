package kubernetes.relationships.workloads

import rego.v1
import data.kubernetes.helpers


# Ingress routes to Service
ingress_routes_to_service contains edge if {
	namespace := input.core.namespaces[ns]
	ingress := namespace.ingresses[_]
	service := namespace.services[_]
	
	backend := ingress.backendServices[_]
	backend == service.name
	
	edge := helpers.create_edge(ingress, service, "RoutesTo")
}

# Assumes all routes to an Ingress are externally accessible.  
# External routes to Ingress
external_routes_to_ingress contains edge if {
	external := input.cluster_scoped.external[_]
	ingress_ns := input.core.namespaces[ns]
	ingress := ingress_ns.ingresses[_]

	edge := helpers.create_edge(external, ingress, "ExternalRoutesTo")
}
