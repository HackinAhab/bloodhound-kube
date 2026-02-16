package kubernetes.relationships.rbac_read_secrets
import rego.v1

import data.kubernetes.helpers

# This made sense in theory but in practice it creates a huge number of edges and isn't very useful.

# read_verbs := ["get", "list", "watch"]

# # ClusterRole with permissions to read secrets
# cluster_role_read_secrets contains edge if {
# 	cluster_role := input.cluster_scoped.clusterrole[_]
# 	perm := cluster_role.properties.perms[_]
# 	key := perm_key(perm)
# 	verbs := perm_verbs(perm)
# 	allows_read(verbs)
# 	secret_sel := secret_key(key)

# 	secret := input.namespaces[ns].secret[_]
# 	secret_matches_key(secret, secret_sel)

# 	edge := helpers.create_edge(cluster_role, secret, "CanRead")
# }

# # Role with permissions to read secrets (namespace-scoped)
# role_read_secrets contains edge if {
# 	namespace := input.namespaces[ns]
# 	role := namespace.role[_]
# 	perm := role.properties.perms[_]
# 	key := perm_key(perm)
# 	verbs := perm_verbs(perm)
# 	allows_read(verbs)
# 	secret_sel := secret_key(key)

# 	secret := namespace.secret[_]
# 	secret_matches_key(secret, secret_sel)

# 	edge := helpers.create_edge(role, secret, "CanRead")
# }

# perm_key(perm) := key if {
# 	parts := split(perm, ":")
# 	key := trim(parts[0], " ")
# }

# perm_verbs(perm) := verbs if {
# 	parts := split(perm, ":")
# 	count(parts) > 1
# 	verbs := [trim(v, " ") | v := split(parts[1], ",")[_]]
# }

# perm_verbs(perm) := [] if {
# 	parts := split(perm, ":")
# 	count(parts) <= 1
# }

# allows_read(verbs) if {
# 	verbs[_] == "*"
# }

# allows_read(verbs) if {
# 	verb := verbs[_]
# 	verb in read_verbs
# }

# secret_key(key) := {"all": true} if {
# 	is_all_resources(key)
# }

# secret_key(key) := {"name": name} if {
# 	parts := split(key, "/")
# 	parts[0] == "secrets"
# 	count(parts) > 1
# 	name := parts[1]
# }

# secret_key(key) := {"name": name} if {
# 	parts := split(key, "/")
# 	count(parts) > 2
# 	parts[1] == "secrets"
# 	name := parts[2]
# }

# secret_key(key) := {"all": true} if {
# 	parts := split(key, "/")
# 	parts[0] == "secrets"
# 	count(parts) == 1
# }

# secret_key(key) := {"all": true} if {
# 	parts := split(key, "/")
# 	parts[1] == "secrets"
# 	count(parts) == 2
# }

# secret_matches_key(secret, secret_key) if {
# 	secret_key.all == true
# }

# secret_matches_key(secret, secret_key) if {
# 	secret_key.name == secret.properties.name
# }

# is_all_resources(key) if {
# 	key == "*"
# }

# is_all_resources(key) if {
# 	endswith(key, "/*")
# }
