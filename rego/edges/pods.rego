package kubernetes.relationships.pods

import rego.v1
import data.kubernetes.helpers

# Pod scheduled on Node
pod_scheduled_on_node contains edge if {
	namespace := input.namespaces[ns]
	pod := namespace.pod[_]
	pod.properties.nodeName != ""
	
	# Find the node
	node := input.cluster_scoped.node[_]
	node.properties.name == pod.properties.nodeName
	
	edge := helpers.create_edge(pod, node, "ScheduledOn")
}

# Pod with dangerous capabilities scheduled on Node
# pod_dangerous_caps_on_node contains edge if {
# 	namespace := input.namespaces[ns]
# 	pod := namespace.pod[_]
# 	pod.properties.nodeName != ""

# 	pod.properties.__private.hasDangerousCaps == true

# 	# Find the node
# 	node := input.cluster_scoped.node[_]
# 	node.properties.name == pod.properties.nodeName
#     description := sprintf("Container in pod %s has dangerous capabilities that could allow for privilege escalation or container escape.", [pod.properties.name])
# 	edge := helpers.create_edge_with_properties(pod, node, "DangerousCaps", {
# 		"Description": description,
# 	})
# }

# Pod uses ServiceAccount
pod_uses_serviceaccount contains edge if {
	namespace := input.namespaces[ns]
	pod := namespace.pod[_]
	sa := namespace.serviceaccount[_]
	
	sa_name := object.get(pod.properties, "serviceAccount", "default")
	sa.properties.name == sa_name
	
	edge := helpers.create_edge(pod, sa, "Uses")
}

# Secret mounted by Pod (via volume)
pod_mounts_secret contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	pod := namespace.pod[_]

	volume := pod.properties.__private.volumes[_]
	volume.type == "secret"
	volume.secretName == secret.properties.name

	edge := helpers.create_edge(pod, secret, "ReferencesSecret")
}

# Secret referenced by Pod environment variables
pod_references_secret_env contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	pod := namespace.pod[_]

	container := pod.properties.__private.containers[_]
	container.hasSecrets == true

	env_source := container.envFrom[_]
	env_source.secretRef
	env_source.secretRef.name == secret.properties.name

	edge := helpers.create_edge(pod, secret, "ReferencesSecret")
}