package kubernetes.relationships.secret

import rego.v1

import data.kubernetes.helpers

# ServiceAccount token Secret belongs to ServiceAccount
service_account_token_belongs_to_account contains edge if {
	secret := input.core.namespaces[ns].secrets[_]
	sa := input.core.namespaces[ns].serviceaccounts[_]

	secret.type == "kubernetes.io/service-account-token"

	sa_secret := sa.secrets[_]
	sa_secret == secret.name

	edge := helpers.create_edge(secret, sa, "LongLivedToken")
}
