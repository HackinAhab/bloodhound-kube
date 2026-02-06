# Workload Relationships
# Defines relationships for Deployments, Pods, and other workload resources

package kubernetes.relationships.workloads

import rego.v1
import data.kubernetes.helpers

# Deployment creates/owns Pods (via label selector matching)
deployment_owns_pod contains edge if {
	namespace := input.namespaces[ns]
	deployment := namespace.deployment[_]
	pod := namespace.pod[_]
	
	# Match pod labels against deployment selector
	helpers.labels_match_selector(pod.properties.labels, deployment.properties.selector)
	
	edge := helpers.create_edge(deployment, pod, "Owns", 7)
}

# Pod scheduled on Node
pod_scheduled_on_node contains edge if {
	namespace := input.namespaces[ns]
	pod := namespace.pod[_]
	pod.properties.node_name != ""
	
	# Find the node
	node := input.cluster_scoped.node[_]
	node.properties.name == pod.properties.node_name
	
	edge := helpers.create_edge(pod, node, "ScheduledOn", 6)
}

# Pod uses ServiceAccount
pod_uses_serviceaccount contains edge if {
	namespace := input.namespaces[ns]
	pod := namespace.pod[_]
	sa := namespace.service_account[_]
	
	sa_name := object.get(pod.properties, "service_account", "default")
	sa.properties.name == sa_name
	
	edge := helpers.create_edge(pod, sa, "Uses", 8)
}

# Deployment uses ServiceAccount (via pod template)
deployment_uses_serviceaccount contains edge if {
	namespace := input.namespaces[ns]
	deployment := namespace.deployment[_]
	sa := namespace.service_account[_]
	
	sa_name := object.get(deployment.properties.pod_template, "service_account", "default")
	sa.properties.name == sa_name
	
	edge := helpers.create_edge(deployment, sa, "Uses", 7)
}

# Service exposes Pods (via label selector)
service_exposes_pod contains edge if {
	namespace := input.namespaces[ns]
	service := namespace.service[_]
	pod := namespace.pod[_]
	
	# Service selector must match pod labels
	service.properties.selector
	count(service.properties.selector) > 0
	helpers.labels_match_selector(pod.properties.labels, service.properties.selector)
	
	edge := helpers.create_edge(service, pod, "Exposes", 7)
}

# Service exposes Deployment (indirect through pod labels)
service_exposes_deployment contains edge if {
	namespace := input.namespaces[ns]
	service := namespace.service[_]
	deployment := namespace.deployment[_]
	
	# Service selector must match deployment selector
	service.properties.selector
	count(service.properties.selector) > 0
	helpers.labels_match_selector(deployment.properties.selector, service.properties.selector)
	
	edge := helpers.create_edge(service, deployment, "Exposes", 6)
}

# Ingress routes to Service
ingress_routes_to_service contains edge if {
	namespace := input.namespaces[ns]
	ingress := namespace.ingress[_]
	service := namespace.service[_]
	
	# Check if any ingress rule references this service
	some rule
	rule := ingress.properties.rules[_]
	some path
	path := rule.paths[_]
	path.backend_service == service.properties.name
	
	edge := helpers.create_edge(ingress, service, "RoutesTo", 7)
}