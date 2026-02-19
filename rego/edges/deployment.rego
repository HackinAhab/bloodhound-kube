package kubernetes.relationships.workloads

import rego.v1
import data.kubernetes.helpers

# Deployment creates/owns Pods (via label selector matching)
deployment_owns_pod contains edge if {
	namespace := input.core.namespaces[ns]
	deployment := namespace.deployments[_]
	pod := namespace.pods[_]
	
	# Match pod labels against deployment selector
	helpers.labels_match_selector(pod.labels_map, deployment.selector_map)
	
	edge := helpers.create_edge(deployment, pod, "ManagedBy")
}
