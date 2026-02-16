package kubernetes.relationships.secret

import rego.v1

import data.kubernetes.helpers

# ServiceAccount token Secret belongs to ServiceAccount
service_account_token_belongs_to_account contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	sa := namespace.serviceaccount[_]

	secret.properties.isServiceAccountToken == true

	sa_secret := sa.properties.secrets[_]
	sa_secret == secret.properties.name

	edge := helpers.create_edge(secret, sa, "LongLivedToken")
}

