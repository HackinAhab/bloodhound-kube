package kubernetes.relationships.workloads

import rego.v1

import data.kubernetes.helpers

# Deployment owns ReplicaSet
deployment_owns_replicaset contains edge if {
	namespace := input.namespaces[ns]
	deployment := namespace.deployment[_]
	replicaset := namespace.replica_set[_]

	owner := replicaset.properties.ownerReferences[_]
	owner.kind == "Deployment"
	owner.name == deployment.properties.name

	edge := helpers.create_edge(deployment, replicaset, "Owns", 5)
}

# ReplicaSet owns Pod
replicaset_owns_pod contains edge if {
	namespace := input.namespaces[ns]
	replicaset := namespace.replica_set[_]
	pod := namespace.pod[_]

	owner := pod.properties.ownerReferences[_]
	owner.kind == "ReplicaSet"
	owner.name == replicaset.properties.name

	edge := helpers.create_edge(replicaset, pod, "Owns", 5)
}

# StatefulSet owns Pod
statefulset_owns_pod contains edge if {
	namespace := input.namespaces[ns]
	statefulset := namespace.stateful_set[_]
	pod := namespace.pod[_]

	owner := pod.properties.ownerReferences[_]
	owner.kind == "StatefulSet"
	owner.name == statefulset.properties.name

	edge := helpers.create_edge(statefulset, pod, "Owns", 5)
}

# DaemonSet owns Pod
daemonset_owns_pod contains edge if {
	namespace := input.namespaces[ns]
	daemonset := namespace.daemon_set[_]
	pod := namespace.pod[_]

	owner := pod.properties.ownerReferences[_]
	owner.kind == "DaemonSet"
	owner.name == daemonset.properties.name

	edge := helpers.create_edge(daemonset, pod, "Owns", 5)
}

# Deployment directly to Pods (transitive via ReplicaSet - for convenience)
deployment_owns_pods contains edge if {
	namespace := input.namespaces[ns]
	deployment := namespace.deployment[_]
	pod := namespace.pod[_]

	# Pod is owned by ReplicaSet which is owned by Deployment
	pod_owner := pod.properties.ownerReferences[_]
	pod_owner.kind == "ReplicaSet"

	# Find the ReplicaSet
	replicaset := namespace.replica_set[_]
	replicaset.properties.name == pod_owner.name

	# Check ReplicaSet is owned by Deployment
	rs_owner := replicaset.properties.ownerReferences[_]
	rs_owner.kind == "Deployment"
	rs_owner.name == deployment.properties.name

	edge := helpers.create_edge(deployment, pod, "Owns", 4)
}
