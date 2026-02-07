package kubernetes.relationships.rbac

import rego.v1

import data.kubernetes.helpers

# ClusterRole → ServiceAccount via ClusterRoleBinding
cluster_role_to_sa_via_binding contains edge if {
	cluster_role := input.cluster_scoped.cluster_role[_]
	namespace := input.namespaces[ns]
	binding := namespace.cluster_role_binding[_]
	sa := namespace.service_account[_]

	binding.properties.roleRef.name == cluster_role.properties.name
	binding.properties.roleRef.kind == "ClusterRole"

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name
	subject.kind == "ServiceAccount"

	edge := helpers.create_edge_via(cluster_role, sa, binding, "HasRole")
}

# Role → ServiceAccount via RoleBinding (namespace-scoped)
role_to_sa_via_binding contains edge if {
	namespace := input.namespaces[ns]
	role := namespace.role[_]
	binding := namespace.role_binding[_]
	sa := namespace.service_account[_]

	binding.properties.roleRef.name == role.properties.name
	binding.properties.roleRef.kind == "Role"

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name

	edge := helpers.create_edge_via(role, sa, binding, "HasRole")
}

# ClusterRoleBinding grants permissions to subjects
cluster_role_binding_to_subject contains edge if {
	binding := input.cluster_scoped.cluster_role_binding[_]
	namespace := input.namespaces[ns]
	sa := namespace.service_account[_]

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name

	edge := helpers.create_edge(binding, sa, "GrantsTo")
}

# RoleBinding grants permissions to subjects
role_binding_to_subject contains edge if {
	namespace := input.namespaces[ns]
	binding := namespace.role_binding[_]
	sa := namespace.service_account[_]

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name

	edge := helpers.create_edge(binding, sa, "GrantsTo")
}

# ClusterRole → ServiceAccount cross-namespace (via any binding type)
cluster_role_to_sa_cross_namespace contains edge if {
	cluster_role := input.cluster_scoped.cluster_role[_]
	namespace := input.namespaces[ns]
	sa := namespace.service_account[_]

	# Check both ClusterRoleBinding and RoleBinding
	binding := namespace.cluster_role_binding[_]
	binding.properties.roleRef.name == cluster_role.properties.name
	binding.properties.roleRef.kind == "ClusterRole"

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name

	edge := helpers.create_edge_via(cluster_role, sa, binding, "HasRole")
}

cluster_role_to_sa_cross_namespace contains edge if {
	cluster_role := input.cluster_scoped.cluster_role[_]
	namespace := input.namespaces[ns]
	sa := namespace.service_account[_]

	# Check RoleBinding referencing ClusterRole
	binding := namespace.role_binding[_]
	binding.properties.roleRef.name == cluster_role.properties.name
	binding.properties.roleRef.kind == "ClusterRole"

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name

	edge := helpers.create_edge_via(cluster_role, sa, binding, "HasRole")
}

# ServiceAccount token Secret belongs to ServiceAccount
service_account_token_belongs_to_account contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	sa := namespace.service_account[_]

	secret.properties.secret_type == "kubernetes.io/service-account-token"
	secret.properties.annotations["kubernetes.io/service-account.name"] == sa.properties.name

	edge := helpers.create_edge(secret, sa, "BelongsTo")
}

# Secret referenced by ServiceAccount
secret_referenced_by_service_account contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	sa := namespace.service_account[_]

	sa_secret := sa.properties.secrets[_]
	sa_secret.name == secret.properties.name

	edge := helpers.create_edge(secret, sa, "ReferencedBy")
}
