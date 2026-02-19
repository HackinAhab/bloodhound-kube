package kubernetes.relationships.workloads

import rego.v1
import data.kubernetes.helpers

# # Service exposes Pods via label selector
# service_exposes_pods contains edge if {
# 	service := input.core.namespaces[ns].services[_]
# 	pod := input.core.namespaces[ns].pods[_]

# 	helpers.is_subset(service.selector_map, pod.labels_map)

# 	edge := helpers.create_edge(service, pod, "RoutesTo")
# }
