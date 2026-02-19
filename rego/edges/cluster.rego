package kubernetes.relationships.cluster

import rego.v1

import data.kubernetes.helpers

# PersistentVolumeClaim used by Pod
persistent_volume_claim_used_by_pod contains edge if {
	namespace := input.core.namespaces[ns]
	pvc := namespace.persistentvolumeclaims[_]
	pod := namespace.pods[_]

	volume := pod.volumes[_]
	volume.pvcName == pvc.name

	edge := helpers.create_edge(pvc, pod, "MountedBy")
}

# PersistentVolume bound to PersistentVolumeClaim
persistent_volume_bound_to_claim contains edge if {
	pv := input.core.cluster.persistentvolumes[_]
	namespace := input.core.namespaces[ns]
	pvc := namespace.persistentvolumeclaims[_]

	claim_ref := object.get(pv, "claimRef", {})
	claim_ref.name == pvc.name
	claim_ref.namespace == ns

	edge := helpers.create_edge(pv, pvc, "BoundTo")
}
