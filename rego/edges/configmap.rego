package kubernetes.relationships.configmap

import rego.v1

import data.kubernetes.helpers

# ConfigMap mounted by Pod (via volume)
configmap_mounted_by_pod contains edge if {
	namespace := input.namespaces[ns]
	cm := namespace.configmap[_]
	pod := namespace.pod[_]

	volume := pod.properties.__private.volumes[_]
	volume.type == "configmap"
	volume.configMapName == cm.properties.name

	edge := helpers.create_edge(cm, pod, "MountedBy")
}

# ConfigMap referenced by Pod environment variables
configmap_env_referenced_by_pod contains edge if {
	namespace := input.namespaces[ns]
	cm := namespace.configmap[_]
	pod := namespace.pod[_]

	container := pod.properties.__private.containers[_]
	env_source := container.envFrom[_]
	env_source.configMapRef
	env_source.configMapRef.name == cm.properties.name

	edge := helpers.create_edge(cm, pod, "EnvVars")
}