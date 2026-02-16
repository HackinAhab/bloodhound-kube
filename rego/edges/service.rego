package kubernetes.relationships.workloads

import rego.v1
import data.kubernetes.helpers

# Service exposes Pods via label selector
service_exposes_pods contains edge if {
	namespace := input.namespaces[ns]
	service := namespace.service[_]
	pod := namespace.pod[_]

	helpers.is_subset(service.properties.__private.selector_map, pod.properties.__private.labels_map)

	edge := helpers.create_edge(service, pod, "RoutesTo")
}