package kubernetes.relationships.rbac

import data.kubernetes.helpers

# ClusterRole → ServiceAccount via ClusterRoleBinding
cluster_role_to_sa_via_binding[edge] {
	cluster_role := input.cluster_scoped.cluster_role[_]
	namespace := input.namespaces[ns]
	binding := namespace.cluster_role_binding[_]
	sa := namespace.service_account[_]

	binding.properties.roleRef.name == cluster_role.properties.name
	binding.properties.roleRef.kind == "ClusterRole"

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name
	subject.kind == "ServiceAccount"

	edge := helpers.create_edge_via(cluster_role, sa, binding, "HasRole", 10)
}

# Role → ServiceAccount via RoleBinding (namespace-scoped)
role_to_sa_via_binding[edge] {
	namespace := input.namespaces[ns]
	role := namespace.role[_]
	binding := namespace.role_binding[_]
	sa := namespace.service_account[_]

	binding.properties.roleRef.name == role.properties.name
	binding.properties.roleRef.kind == "Role"

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name

	edge := helpers.create_edge_via(role, sa, binding, "HasRole", 10)
}

# ClusterRoleBinding grants permissions to subjects
cluster_role_binding_to_subject[edge] {
	binding := input.cluster_scoped.cluster_role_binding[_]
	namespace := input.namespaces[ns]
	sa := namespace.service_account[_]

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name

	edge := helpers.create_edge(binding, sa, "GrantsTo", 9)
}

# RoleBinding grants permissions to subjects
role_binding_to_subject[edge] {
	namespace := input.namespaces[ns]
	binding := namespace.role_binding[_]
	sa := namespace.service_account[_]

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name

	edge := helpers.create_edge(binding, sa, "GrantsTo", 9)
}

# ClusterRole → ServiceAccount cross-namespace (via any binding type)
cluster_role_to_sa_cross_namespace[edge] {
	cluster_role := input.cluster_scoped.cluster_role[_]
	namespace := input.namespaces[ns]
	sa := namespace.service_account[_]

	# Check both ClusterRoleBinding and RoleBinding
	binding := namespace.cluster_role_binding[_]
	binding.properties.roleRef.name == cluster_role.properties.name
	binding.properties.roleRef.kind == "ClusterRole"

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name

	edge := helpers.create_edge_via(cluster_role, sa, binding, "HasRole", 9)
}

cluster_role_to_sa_cross_namespace[edge] {
	cluster_role := input.cluster_scoped.cluster_role[_]
	namespace := input.namespaces[ns]
	sa := namespace.service_account[_]

	# Check RoleBinding referencing ClusterRole
	binding := namespace.role_binding[_]
	binding.properties.roleRef.name == cluster_role.properties.name
	binding.properties.roleRef.kind == "ClusterRole"

	subject := binding.properties.subjects[_]
	subject.name == sa.properties.name

	edge := helpers.create_edge_via(cluster_role, sa, binding, "HasRole", 9)
}

# ServiceAccount token Secret belongs to ServiceAccount
service_account_token_belongs_to_account[edge] {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	sa := namespace.service_account[_]

	secret.properties.secret_type == "kubernetes.io/service-account-token"
	secret.properties.annotations["kubernetes.io/service-account.name"] == sa.properties.name

	edge := helpers.create_edge(secret, sa, "BelongsTo", 9)
}

# Secret referenced by ServiceAccount
secret_referenced_by_service_account[edge] {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	sa := namespace.service_account[_]

	sa_secret := sa.properties.secrets[_]
	sa_secret.name == secret.properties.name

	edge := helpers.create_edge(secret, sa, "ReferencedBy", 8)
}
