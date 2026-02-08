# External Secrets Relationships
# Links ExternalSecret resources to their SecretStore/ClusterSecretStore

package kubernetes.relationships.external_secrets

import rego.v1
import data.kubernetes.helpers

# ExternalSecret uses SecretStore (namespaced)
external_secret_uses_secretstore contains edge if {
	namespace := input.namespaces[ns]
	external_secret := namespace.externalsecret[_]
	secret_store := namespace.secretstore[_]

	store_name := object.get(external_secret.properties, "store_name", "")
	store_name != ""

	store_kind := lower(object.get(external_secret.properties, "store_kind", "SecretStore"))
	store_kind == "secretstore"
	secret_store.properties.name == store_name

	edge := helpers.create_edge(external_secret, secret_store, "Uses")
}

# ExternalSecret uses ClusterSecretStore (cluster-scoped)
external_secret_uses_clustersecretstore contains edge if {
	namespace := input.namespaces[ns]
	external_secret := namespace.externalsecret[_]
	cluster_store := input.cluster_scoped.clustersecretstore[_]

	store_name := object.get(external_secret.properties, "store_name", "")
	store_name != ""

	store_kind := lower(object.get(external_secret.properties, "store_kind", "SecretStore"))
	store_kind == "clustersecretstore"
	cluster_store.properties.name == store_name

	edge := helpers.create_edge(external_secret, cluster_store, "Uses")
}

# ExternalSecret manages Secret (explicit target name)
external_secret_manages_secret contains edge if {
	namespace := input.namespaces[ns]
	external_secret := namespace.externalsecret[_]
	secret := namespace.secret[_]

	target_name := object.get(external_secret.properties, "target_name", "")
	target_name != ""
	secret.properties.name == target_name

	edge := helpers.create_edge(secret,external_secret, "ManagedBy")
}

# ExternalSecret manages Secret (default target name)
external_secret_manages_secret_default contains edge if {
	namespace := input.namespaces[ns]
	external_secret := namespace.externalsecret[_]
	secret := namespace.secret[_]

	target_name := object.get(external_secret.properties, "target_name", "")
	target_name == ""
	secret.properties.name == external_secret.properties.name

	edge := helpers.create_edge(secret, external_secret, "ManagedBy")
}
