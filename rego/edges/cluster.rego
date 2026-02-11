package kubernetes.relationships.cluster

import rego.v1

import data.kubernetes.helpers

# PersistentVolumeClaim used by Pod
persistent_volume_claim_used_by_pod contains edge if {
	namespace := input.namespaces[ns]
	pvc := namespace.persistentvolumeclaim[_]
	pod := namespace.pod[_]

	volume := pod.properties.__private.volumes[_]
	volume.pvcName == pvc.properties.name

	edge := helpers.create_edge(pvc, pod, "UsedBy")
}

# PersistentVolume bound to PersistentVolumeClaim
persistent_volume_bound_to_claim contains edge if {
	pv := input.cluster_scoped.persistentvolume[_]
	namespace := input.namespaces[ns]
	pvc := namespace.persistentvolumeclaim[_]

	claim_ref := object.get(pv.properties, "claimRef", {})
	claim_ref.name == pvc.properties.name
	claim_ref.namespace == ns

	edge := helpers.create_edge(pv, pvc, "BoundTo")
}
