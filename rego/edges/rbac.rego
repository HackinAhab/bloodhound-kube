package kubernetes.relationships.rbac

import rego.v1

import data.kubernetes.helpers

# ClusterRole → ServiceAccount via ClusterRoleBinding
cluster_role_to_sa_via_binding contains edge if {
	cluster_role := input.cluster_scoped.clusterrole[_]
	binding := input.cluster_scoped.clusterrolebinding[_]
	namespace := input.namespaces[ns]
	sa := namespace.serviceaccount[_]

	binding.properties.role_name == cluster_role.properties.name
	binding.properties.role_kind == "ClusterRole"

	subject := binding.properties.subjects[_]
	subject.kind == "ServiceAccount"
	subject.name == sa.properties.name
	object.get(subject, "namespace", "") == sa.properties.namespace

	edge := helpers.create_edge_via(cluster_role, sa, binding, "HasRole")
}

# Role → ServiceAccount via RoleBinding (namespace-scoped)
role_to_sa_via_binding contains edge if {
	namespace := input.namespaces[ns]
	role := namespace.role[_]
	binding := namespace.rolebinding[_]
	sa := namespace.serviceaccount[_]

	binding.properties.role_name == role.properties.name
	binding.properties.role_kind == "Role"

	subject := binding.properties.subjects[_]
	subject.kind == "ServiceAccount"
	subject.name == sa.properties.name
	object.get(subject, "namespace", binding.properties.namespace) == sa.properties.namespace

	edge := helpers.create_edge_via(role, sa, binding, "HasRole")
}

# ClusterRoleBinding grants permissions to subjects
cluster_role_binding_to_subject contains edge if {
	binding := input.cluster_scoped.clusterrolebinding[_]
	namespace := input.namespaces[ns]
	sa := namespace.serviceaccount[_]

	subject := binding.properties.subjects[_]
	subject.kind == "ServiceAccount"
	subject.name == sa.properties.name
	object.get(subject, "namespace", "") == sa.properties.namespace

	edge := helpers.create_edge(binding, sa, "GrantsTo")
}

# RoleBinding grants permissions to subjects
role_binding_to_subject contains edge if {
	namespace := input.namespaces[ns]
	binding := namespace.rolebinding[_]
	sa := namespace.serviceaccount[_]

	subject := binding.properties.subjects[_]
	subject.kind == "ServiceAccount"
	subject.name == sa.properties.name
	object.get(subject, "namespace", binding.properties.namespace) == sa.properties.namespace

	edge := helpers.create_edge(binding, sa, "GrantsTo")
}

# ClusterRole → ServiceAccount cross-namespace (via any binding type)
cluster_role_to_sa_cross_namespace contains edge if {
	cluster_role := input.cluster_scoped.clusterrole[_]
	binding := input.cluster_scoped.clusterrolebinding[_]
	namespace := input.namespaces[ns]
	sa := namespace.serviceaccount[_]

	# Check ClusterRoleBinding
	binding.properties.role_name == cluster_role.properties.name
	binding.properties.role_kind == "ClusterRole"

	subject := binding.properties.subjects[_]
	subject.kind == "ServiceAccount"
	subject.name == sa.properties.name
	object.get(subject, "namespace", "") == sa.properties.namespace

	edge := helpers.create_edge_via(cluster_role, sa, binding, "HasRole")
}

cluster_role_to_sa_cross_namespace contains edge if {
	cluster_role := input.cluster_scoped.clusterrole[_]
	namespace := input.namespaces[ns]
	sa := namespace.serviceaccount[_]

	# Check RoleBinding referencing ClusterRole
	binding := namespace.rolebinding[_]
	binding.properties.role_name == cluster_role.properties.name
	binding.properties.role_kind == "ClusterRole"

	subject := binding.properties.subjects[_]
	subject.kind == "ServiceAccount"
	subject.name == sa.properties.name
	object.get(subject, "namespace", binding.properties.namespace) == sa.properties.namespace

	edge := helpers.create_edge_via(cluster_role, sa, binding, "HasRole")
}

# ServiceAccount token Secret belongs to ServiceAccount
service_account_token_belongs_to_account contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	sa := namespace.serviceaccount[_]

	secret.properties.secret_type == "kubernetes.io/service-account-token"
	secret.properties.annotations["kubernetes.io/service-account.name"] == sa.properties.name

	edge := helpers.create_edge(secret, sa, "BelongsTo")
}

# Secret referenced by ServiceAccount
secret_referenced_by_service_account contains edge if {
	namespace := input.namespaces[ns]
	secret := namespace.secret[_]
	sa := namespace.serviceaccount[_]

	sa_secret := sa.properties.secrets[_]
	sa_secret.name == secret.properties.name

	edge := helpers.create_edge(secret, sa, "ReferencedBy")
}
