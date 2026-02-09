package kubernetes.relationships.storage

import rego.v1

import data.kubernetes.helpers

# Secret mounted by Pod (via volume)
secret_mounted_by_pod contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	pod := namespace.pod[_]

	volume := pod.properties.__private.volumes[_]
	volume.type == "secret"
	volume.secretName == secret.properties.name

	edge := helpers.create_edge(secret, pod, "MountedBy")
}

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

# Secret referenced by Pod environment variables
secret_env_referenced_by_pod contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	pod := namespace.pod[_]

	container := pod.properties.__private.containers[_]
	container.hasSecrets == true

	env_source := container.envFrom[_]
	env_source.secretRef
	env_source.secretRef.name == secret.properties.name

	edge := helpers.create_edge(secret, pod, "ReferencedBy")
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

	edge := helpers.create_edge(cm, pod, "ReferencedBy")
}

# TLS Secret used by Ingress
tls_secret_used_by_ingress contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	ingress := namespace.ingress[_]

	secret.properties.isTlsSecret == true

	tls_config := ingress.properties.tls[_]
	tls_config.secretName == secret.properties.name

	edge := helpers.create_edge(secret, ingress, "SecuresWith")
}

# ServiceAccount token Secret belongs to ServiceAccount
service_account_token_belongs_to_account contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	sa := namespace.serviceaccount[_]

	secret.properties.isServiceAccountToken == true

	sa_secret := sa.properties.secrets[_]
	sa_secret == secret.properties.name

	edge := helpers.create_edge(secret, sa, "BelongsTo")
}

# Pod uses ServiceAccount
pod_uses_service_account contains edge if {
	namespace := input.namespaces[ns]
	pod := namespace.pod[_]
	sa := namespace.serviceaccount[_]

	pod.properties.serviceAccount == sa.properties.name

	edge := helpers.create_edge(pod, sa, "Uses")
}
