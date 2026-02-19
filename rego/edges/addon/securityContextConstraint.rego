package kubernetes.relationships.scc

import rego.v1

import data.kubernetes.helpers

# Pod attached to SecurityContextConstraints via annotation
pod_attached_to_scc contains edge if {
	namespace := input.core.namespaces[ns]
	pod := namespace.pods[_]

	scc := input.core.cluster.securitycontextconstraints[_]

	scc_name := pod.annotations_map["openshift.io/scc"]
	scc_name != ""
	scc.name == scc_name

	edge := helpers.create_edge(scc, pod, "EnforcedSCC")
}
