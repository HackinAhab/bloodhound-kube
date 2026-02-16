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