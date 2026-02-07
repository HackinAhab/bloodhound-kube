# Storage Relationships
# Defines relationships for Secrets, ConfigMaps, and volume usage

package kubernetes.relationships.storage_advanced

import rego.v1
import data.kubernetes.helpers

# Secret mounted by Pod (via volume)
secret_mounted_by_pod contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	pod := namespace.pod[_]
	
	# Check if pod has volume referencing this secret
	volume := pod.properties.volumes[_]
	volume.type == "secret"
	volume.secret_name == secret.properties.name
	
	edge := helpers.create_edge(secret, pod, "MountedBy")
}

# ConfigMap mounted by Pod (via volume)
configmap_mounted_by_pod contains edge if {
	namespace := input.namespaces[ns]
	cm := namespace.config_map[_]
	pod := namespace.pod[_]
	
	# Check if pod has volume referencing this configmap
	volume := pod.properties.volumes[_]
	volume.type == "configmap"
	volume.configmap_name == cm.properties.name
	
	edge := helpers.create_edge(cm, pod, "MountedBy")
}

# Secret referenced by Pod environment variables
secret_env_referenced_by_pod contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	pod := namespace.pod[_]
	
	# Check containers for secret references
	container := pod.properties.containers[_]
	container.has_secrets == true
	
	# Additional check via env_from
	env_source := container.env_from[_]
	env_source.secret_ref
	env_source.secret_ref.name == secret.properties.name
	
	edge := helpers.create_edge(secret, pod, "ReferencedBy")
}

# ConfigMap referenced by Pod environment variables
configmap_env_referenced_by_pod contains edge if {
	namespace := input.namespaces[ns]
	cm := namespace.config_map[_]
	pod := namespace.pod[_]
	
	# Check containers for configmap references
	container := pod.properties.containers[_]
	env_source := container.env_from[_]
	env_source.configmap_ref
	env_source.configmap_ref.name == cm.properties.name
	
	edge := helpers.create_edge(cm, pod, "ReferencedBy")
}

# TLS Secret used by Ingress
tls_secret_used_by_ingress contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	ingress := namespace.ingress[_]
	
	secret.properties.is_tls_secret == true
	
	# Check TLS configuration
	tls_config := ingress.properties.tls[_]
	tls_config.secret_name == secret.properties.name
	
	edge := helpers.create_edge(secret, ingress, "SecuresWith")
}

# ServiceAccount token Secret belongs to ServiceAccount
service_account_token_belongs_to_account contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	sa := namespace.service_account[_]
	
	secret.properties.is_service_account_token == true
	
	# Check if secret is referenced by service account
	sa_secret := sa.properties.secrets[_]
	sa_secret == secret.properties.name
	
	edge := helpers.create_edge(secret, sa, "BelongsTo")
}
