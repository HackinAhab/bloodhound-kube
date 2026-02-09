package kubernetes.relationships.networking

import rego.v1

import data.kubernetes.helpers

# Service exposes Pods via label selector
service_exposes_pods contains edge if {
	namespace := input.namespaces[ns]
	service := namespace.service[_]
	pod := namespace.pod[_]

	helpers.is_subset(service.properties.selector, pod.properties.labels_map)

	edge := helpers.create_edge(service, pod, "Exposes")
}

# Service exposes Deployment via label selector
service_exposes_deployment contains edge if {
	namespace := input.namespaces[ns]
	service := namespace.service[_]
	deployment := namespace.deployment[_]

	labels := object.get(deployment.properties, ["pod_template", "labels"], deployment.properties.selector)
	helpers.is_subset(service.properties.selector, labels)

	edge := helpers.create_edge(service, deployment, "Exposes")
}

# Ingress routes to Service
ingress_routes_to_service contains edge if {
	namespace := input.namespaces[ns]
	ingress := namespace.ingress[_]
	service := namespace.service[_]

	# Check rules for backend service references
	rule := ingress.properties.rules[_]
	path := rule.paths[_]

	path.backend_service == service.properties.name

	edge := helpers.create_edge(ingress, service, "RoutesTo")
}

# Ingress routes to Service (legacy API with serviceName)
ingress_routes_to_service_v1 contains edge if {
	namespace := input.namespaces[ns]
	ingress := namespace.ingress[_]
	service := namespace.service[_]

	# Check rules for backend service references (legacy API)
	rule := ingress.properties.rules[_]
	path := rule.http.paths[_]
	backend := path.backend

	backend.serviceName == service.properties.name

	edge := helpers.create_edge(ingress, service, "RoutesTo")
}

# Ingress routes to Service (raw v1 API)
ingress_routes_to_service_v1 contains edge if {
	namespace := input.namespaces[ns]
	ingress := namespace.ingress[_]
	service := namespace.service[_]

	# Check rules for backend service references (v1 API)
	rule := ingress.properties.rules[_]
	path := rule.http.paths[_]
	backend := path.backend

	backend.service.name == service.properties.name

	edge := helpers.create_edge(ingress, service, "RoutesTo")
}

# NetworkPolicy applies to Pods
network_policy_applies_to_pods contains edge if {
	namespace := input.namespaces[ns]
	netpol := namespace.networkpolicy[_]
	pod := namespace.pod[_]

	helpers.is_subset(netpol.properties.pod_selector, pod.properties.labels_map)

	edge := helpers.create_edge(netpol, pod, "AppliesTo")
}
