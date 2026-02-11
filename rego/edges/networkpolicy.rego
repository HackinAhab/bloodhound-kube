package kubernetes.relationships.networkpolicy

import rego.v1

import data.kubernetes.helpers


# NetworkPolicy applies to Pods
network_policy_applies_to_pods contains edge if {
	namespace := input.namespaces[ns]
	netpol := namespace.networkpolicy[_]
	pod := namespace.pod[_]

	helpers.is_subset(netpol.properties.podSelector, pod.properties.__private.labels_map)

	edge := helpers.create_edge(netpol, pod, "AppliesTo")
}
