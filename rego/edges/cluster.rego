package kubernetes.relationships.cluster

import rego.v1

import data.kubernetes.helpers

# Pod scheduled on Node
pod_scheduled_on_node contains edge if {
	node := input.cluster_scoped.node[_]
	namespace := input.namespaces[ns]
	pod := namespace.pod[_]

	pod.properties.nodeName == node.properties.name

	edge := helpers.create_edge(pod, node, "ScheduledOn")
}

# PersistentVolumeClaim used by Pod
persistent_volume_claim_used_by_pod contains edge if {
	namespace := input.namespaces[ns]
	pvc := namespace.persistent_volume_claim[_]
	pod := namespace.pod[_]

	volume := pod.properties.volumes[_]
	volume.persistentVolumeClaim.claimName == pvc.properties.name

	edge := helpers.create_edge(pvc, pod, "UsedBy")
}

# PersistentVolume bound to PersistentVolumeClaim
persistent_volume_bound_to_claim contains edge if {
	pv := input.cluster_scoped.persistent_volume[_]
	namespace := input.namespaces[ns]
	pvc := namespace.persistent_volume_claim[_]

	pv.properties.claimRef.name == pvc.properties.name
	pv.properties.claimRef.namespace == namespace

	edge := helpers.create_edge(pv, pvc, "BoundTo")
}
