# External Secrets Relationships
# Links ExternalSecret resources to their SecretStore/ClusterSecretStore

package kubernetes.relationships.external_secrets

import rego.v1
import data.kubernetes.helpers

# ExternalSecret uses SecretStore (namespaced)
external_secret_uses_secretstore contains edge if {
	namespace := input.core.namespaces[ns]
	external_secret := namespace.externalsecrets[_]
	secret_store := namespace.secretstores[_]

	store_name := object.get(external_secret, "storeName", "")
	store_name != ""

	store_kind := lower(object.get(external_secret, "storeKind", "SecretStore"))
	store_kind == "secretstore"
	secret_store.name == store_name

	edge := helpers.create_edge(external_secret, secret_store, "ManagedBy")
}

# ExternalSecret uses ClusterSecretStore (cluster-scoped)
external_secret_uses_clustersecretstore contains edge if {
	namespace := input.core.namespaces[ns]
	external_secret := namespace.externalsecrets[_]
	cluster_store := input.core.cluster.clustersecretstores[_]

	store_name := object.get(external_secret, "storeName", "")
	store_name != ""

	store_kind := lower(object.get(external_secret, "storeKind", "SecretStore"))
	store_kind == "clustersecretstore"
	cluster_store.name == store_name

	edge := helpers.create_edge(external_secret, cluster_store, "ManagedBy")
}

# ExternalSecret manages Secret (explicit target name)
external_secret_manages_secret contains edge if {
	namespace := input.core.namespaces[ns]
	external_secret := namespace.externalsecrets[_]
	secret := namespace.secrets[_]

	target_name := object.get(external_secret, "targetName", "")
	target_name != ""
	secret.name == target_name

	edge := helpers.create_edge(secret,external_secret, "ManagedBy")
}

# ExternalSecret manages Secret (default target name)
external_secret_manages_secret_default contains edge if {
	namespace := input.core.namespaces[ns]
	external_secret := namespace.externalsecrets[_]
	secret := namespace.secrets[_]

	target_name := object.get(external_secret, "targetName", "")
	target_name == ""
	secret.name == external_secret.name

	edge := helpers.create_edge(secret, external_secret, "ManagedBy")
}
