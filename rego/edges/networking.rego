package kubernetes.relationships.networking

import rego.v1

import data.kubernetes.helpers

# Service exposes Pods via label selector
service_exposes_pods contains edge if {
	namespace := input.namespaces[ns]
	service := namespace.service[_]
	pod := namespace.pod[_]

	helpers.is_subset(service.properties.__private.selector_map, pod.properties.__private.labels_map)

	edge := helpers.create_edge(service, pod, "Exposes")
}

# Service exposes Deployment via label selector
service_exposes_deployment contains edge if {
	namespace := input.namespaces[ns]
	service := namespace.service[_]
	deployment := namespace.deployment[_]

	labels := object.get(deployment.properties, ["pod_template", "__private", "labels_map"], deployment.properties.__private.selector_map)
	helpers.is_subset(service.properties.__private.selector_map, labels)

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

	path.backendService == service.properties.name

	edge := helpers.create_edge(ingress, service, "RoutesTo")
}


# NetworkPolicy applies to Pods
network_policy_applies_to_pods contains edge if {
	namespace := input.namespaces[ns]
	netpol := namespace.networkpolicy[_]
	pod := namespace.pod[_]

	helpers.is_subset(netpol.properties.podSelector, pod.properties.__private.labels_map)

	edge := helpers.create_edge(netpol, pod, "AppliesTo")
}
