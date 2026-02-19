package kubernetes.relationships.pods

import rego.v1
import data.kubernetes.helpers

# Pod scheduled on Node
pod_scheduled_on_node contains edge if {
	pod := input.core.namespaces[ns].pods[_]
	pod.nodeName != ""

	node := input.core.cluster.nodes[_]
	node.name == pod.nodeName

	edge := helpers.create_edge(pod, node, "ScheduledOn")
}

# Pod uses ServiceAccount
pod_uses_serviceaccount contains edge if {
	pod := input.core.namespaces[ns].pods[_]
	sa := input.core.namespaces[ns].serviceaccounts[_]

	sa_name := object.get(pod, "serviceAccount", "default")
	sa_name != ""
	sa_name != "default"
	sa.name == sa_name

	edge := helpers.create_edge(pod, sa, "Uses")
}

# Secret mounted by Pod (via volume)
pod_mounts_secret contains edge if {
	secret := input.core.namespaces[ns].secrets[_]
	pod := input.core.namespaces[ns].pods[_]

	volume := pod.volumes[_]
	volume.type == "secret"
	volume.secretName == secret.name

	edge := helpers.create_edge(secret, pod, "MountedBy")
}

# Secret referenced by Pod environment variables
pod_references_secret_env contains edge if {
	secret := input.core.namespaces[ns].secrets[_]
	pod := input.core.namespaces[ns].pods[_]

	container := pod.containers[_]
	env_source := container.envFrom[_]
	env_source.secretRef
	env_source.secretRef.name == secret.name

	edge := helpers.create_edge(secret, pod, "EnvVars")
}
