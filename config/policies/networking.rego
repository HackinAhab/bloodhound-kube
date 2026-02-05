package kubernetes.relationships.networking

import rego.v1

import data.kubernetes.helpers

# Service exposes Pods via label selector
service_exposes_pods contains edge if {
	namespace := input.namespaces[ns]
	service := namespace.service[_]
	pod := namespace.pod[_]

	helpers.is_subset(service.properties.selector, pod.properties.labels)

	edge := helpers.create_edge(service, pod, "Exposes", 7)
}

# Service exposes Deployment via label selector
service_exposes_deployment contains edge if {
	namespace := input.namespaces[ns]
	service := namespace.service[_]
	deployment := namespace.deployment[_]

	helpers.is_subset(service.properties.selector, deployment.properties.spec.template.metadata.labels)

	edge := helpers.create_edge(service, deployment, "Exposes", 7)
}

# Ingress routes to Service
ingress_routes_to_service contains edge if {
	namespace := input.namespaces[ns]
	ingress := namespace.ingress[_]
	service := namespace.service[_]

	# Check rules for backend service references
	rule := ingress.properties.spec.rules[_]
	path := rule.http.paths[_]
	backend := path.backend

	backend.serviceName == service.properties.name

	edge := helpers.create_edge(ingress, service, "RoutesTo", 6)
}

# Ingress routes to Service (newer API with service.name)
ingress_routes_to_service_v1 contains edge if {
	namespace := input.namespaces[ns]
	ingress := namespace.ingress[_]
	service := namespace.service[_]

	# Check rules for backend service references (v1 API)
	rule := ingress.properties.spec.rules[_]
	path := rule.http.paths[_]
	backend := path.backend

	backend.service.name == service.properties.name

	edge := helpers.create_edge(ingress, service, "RoutesTo", 6)
}

# NetworkPolicy applies to Pods
network_policy_applies_to_pods contains edge if {
	namespace := input.namespaces[ns]
	netpol := namespace.network_policy[_]
	pod := namespace.pod[_]

	helpers.is_subset(netpol.properties.podSelector, pod.properties.labels)

	edge := helpers.create_edge(netpol, pod, "AppliesTo", 6)
}
