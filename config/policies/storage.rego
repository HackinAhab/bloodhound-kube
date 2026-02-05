package kubernetes.relationships.storage

import rego.v1

import data.kubernetes.helpers

# Secret mounted by Pod
secret_mounted_by_pod contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	pod := namespace.pod[_]

	volume := pod.properties.volumes[_]
	volume.secret.secretName == secret.properties.name

	edge := helpers.create_edge(secret, pod, "MountedBy", 8)
}

# ConfigMap used by Pod
configmap_used_by_pod contains edge if {
	namespace := input.namespaces[ns]
	cm := namespace.config_map[_]
	pod := namespace.pod[_]

	volume := pod.properties.volumes[_]
	volume.configMap.name == cm.properties.name

	edge := helpers.create_edge(cm, pod, "UsedBy", 8)
}

# Secret referenced by Deployment environment variables
secret_referenced_by_deployment contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	deployment := namespace.deployment[_]

	container := deployment.properties.spec.template.spec.containers[_]
	env := container.env[_]
	env.valueFrom.secretKeyRef.name == secret.properties.name

	edge := helpers.create_edge(secret, deployment, "ReferencedBy", 8)
}

# ConfigMap referenced by Deployment environment variables
configmap_referenced_by_deployment contains edge if {
	namespace := input.namespaces[ns]
	cm := namespace.config_map[_]
	deployment := namespace.deployment[_]

	container := deployment.properties.spec.template.spec.containers[_]
	env := container.env[_]
	env.valueFrom.configMapKeyRef.name == cm.properties.name

	edge := helpers.create_edge(cm, deployment, "ReferencedBy", 8)
}

# TLS Secret used by Ingress
tls_secret_used_by_ingress contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	ingress := namespace.ingress[_]

	secret.properties.secret_type == "kubernetes.io/tls"

	tls_config := ingress.properties.tls[_]
	tls_config.secretName == secret.properties.name

	edge := helpers.create_edge(secret, ingress, "SecuresWith", 8)
}

# Pod uses ServiceAccount
pod_uses_service_account contains edge if {
	namespace := input.namespaces[ns]
	pod := namespace.pod[_]
	sa := namespace.service_account[_]

	pod.properties.spec.serviceAccountName == sa.properties.name

	edge := helpers.create_edge(pod, sa, "Uses", 7)
}
