package kubernetes.relationships.workloads

import rego.v1
import data.kubernetes.helpers

# Deployment creates/owns Pods (via label selector matching)
deployment_owns_pod contains edge if {
	namespace := input.namespaces[ns]
	deployment := namespace.deployment[_]
	pod := namespace.pod[_]
	
	# Match pod labels against deployment selector
	helpers.labels_match_selector(pod.properties.__private.labels_map, deployment.properties.__private.selector_map)
	
	edge := helpers.create_edge(deployment, pod, "ManagedBy")
}

# Deployment uses ServiceAccount (via pod template)
deployment_uses_serviceaccount contains edge if {
	namespace := input.namespaces[ns]
	deployment := namespace.deployment[_]
	sa := namespace.serviceaccount[_]
	
	sa_name := object.get(object.get(deployment.properties, "podTemplate", deployment.properties.pod_template), "serviceAccount", "default")
	sa.properties.name == sa_name
	
	edge := helpers.create_edge(deployment, sa, "Uses")
}