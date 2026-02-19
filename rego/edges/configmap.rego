package kubernetes.relationships.configmap

import rego.v1

import data.kubernetes.helpers

# ConfigMap mounted by Pod (via volume)
configmap_mounted_by_pod contains edge if {
	cm := input.core.namespaces[ns].configmaps[_]
	pod := input.core.namespaces[ns].pods[_]

	volume := pod.volumes[_]
	volume.type == "configmap"
	volume.configMapName == cm.name

	edge := helpers.create_edge(cm, pod, "MountedBy")
}

# ConfigMap referenced by Pod environment variables
configmap_env_referenced_by_pod contains edge if {
	cm := input.core.namespaces[ns].configmaps[_]
	pod := input.core.namespaces[ns].pods[_]

	container := pod.containers[_]
	env_source := container.envFrom[_]
	env_source.configMapRef
	env_source.configMapRef.name == cm.name

	edge := helpers.create_edge(cm, pod, "EnvVars")
}
