package kubernetes.relationships.rbac

import rego.v1

import data.kubernetes.helpers

# ClusterRole → ServiceAccount via ClusterRoleBinding
cluster_role_to_sa_via_binding contains edge if {
	cluster_role := input.core.cluster.clusterroles[_]
	binding := input.core.cluster.clusterrolebindings[_]
	namespace := input.core.namespaces[ns]
	sa := namespace.serviceaccounts[_]

	binding.roleName == cluster_role.name
	binding.roleKind == "ClusterRole"

	subject := binding.subjects[_]
	subject.kind == "ServiceAccount"
	subject.name == sa.name
	object.get(subject, "namespace", "") == sa.namespace

	edge := helpers.create_edge_via(cluster_role, sa, binding, "PermissionsFromRole")
}

# Role → ServiceAccount via RoleBinding (namespace-scoped)
role_to_sa_via_binding contains edge if {
	namespace := input.core.namespaces[ns]
	role := namespace.roles[_]
	binding := namespace.rolebindings[_]
	sa := namespace.serviceaccounts[_]

	binding.roleName == role.name
	binding.roleKind == "Role"

	subject := binding.subjects[_]
	subject.kind == "ServiceAccount"
	subject.name == sa.name
	object.get(subject, "namespace", binding.namespace) == sa.namespace

	edge := helpers.create_edge_via(role, sa, binding, "PermissionsFromRole")
}

# # ClusterRoleBinding grants permissions to subjects
# cluster_role_binding_to_subject contains edge if {
# 	binding := input.core.cluster.clusterrolebindings[_]
# 	namespace := input.core.namespaces[ns]
# 	sa := namespace.serviceaccounts[_]

# 	subject := binding.subjects[_]
# 	subject.kind == "ServiceAccount"
# 	subject.name == sa.name
# 	object.get(subject, "namespace", "") == sa.namespace

# 	edge := helpers.create_edge(binding, sa, "GrantsTo")
# }

# RoleBinding grants permissions to subjects
# role_binding_to_subject contains edge if {
# 	namespace := input.core.namespaces[ns]
# 	binding := namespace.rolebindings[_]
# 	sa := namespace.serviceaccounts[_]

# 	subject := binding.subjects[_]
# 	subject.kind == "ServiceAccount"
# 	subject.name == sa.name
# 	object.get(subject, "namespace", binding.namespace) == sa.namespace

# 	edge := helpers.create_edge(binding, sa, "GrantsTo")
# }

# ClusterRole → ServiceAccount cross-namespace (via any binding type)
cluster_role_to_sa_cross_namespace contains edge if {
	cluster_role := input.core.cluster.clusterroles[_]
	binding := input.core.cluster.clusterrolebindings[_]
	namespace := input.core.namespaces[ns]
	sa := namespace.serviceaccounts[_]

	# Check ClusterRoleBinding
	binding.roleName == cluster_role.name
	binding.roleKind == "ClusterRole"

	subject := binding.subjects[_]
	subject.kind == "ServiceAccount"
	subject.name == sa.name
	object.get(subject, "namespace", "") == sa.namespace

	edge := helpers.create_edge_via(cluster_role, sa, binding, "PermissionsFromRole")
}

cluster_role_to_sa_cross_namespace contains edge if {
	cluster_role := input.core.cluster.clusterroles[_]
	namespace := input.core.namespaces[ns]
	sa := namespace.serviceaccounts[_]

	# Check RoleBinding referencing ClusterRole
	binding := namespace.rolebindings[_]
	binding.roleName == cluster_role.name
	binding.roleKind == "ClusterRole"

	subject := binding.subjects[_]
	subject.kind == "ServiceAccount"
	subject.name == sa.name
	object.get(subject, "namespace", binding.namespace) == sa.namespace

	edge := helpers.create_edge_via(cluster_role, sa, binding, "PermissionsFromRole")
}

# ServiceAccount token Secret belongs to ServiceAccount
service_account_token_belongs_to_account contains edge if {
	namespace := input.core.namespaces[ns]
	secret := namespace.secrets[_]
	sa := namespace.serviceaccounts[_]

	secret.type == "kubernetes.io/service-account-token"
	secret.annotations_map["kubernetes.io/service-account.name"] == sa.name

	edge := helpers.create_edge(sa, secret, "SAToken")
}

# # Secret referenced by ServiceAccount
# secret_referenced_by_service_account contains edge if {
# 	namespace := input.core.namespaces[ns]
# 	secret := namespace.secrets[_]
# 	sa := namespace.serviceaccounts[_]

# 	sa_secret := sa.secrets[_]
# 	sa_secret == secret.name

# 	edge := helpers.create_edge(secret, sa, "ReferencedBy")
# }
