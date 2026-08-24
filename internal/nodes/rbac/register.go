package rbac

import (
	"bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func Register() {
	framework.RegisterResources(
		framework.ResourceRegistration{
			GVK:          rbacv1.SchemeGroupVersion.WithKind("Role"),
			TypedBuilder: BuildRoleNode,
		},
		framework.ResourceRegistration{
			GVK:          rbacv1.SchemeGroupVersion.WithKind("RoleBinding"),
			TypedBuilder: BuildRoleBindingNode,
		},
		framework.ResourceRegistration{
			GVK:          rbacv1.SchemeGroupVersion.WithKind("ClusterRole"),
			TypedBuilder: BuildClusterRoleNode,
		},
		framework.ResourceRegistration{
			GVK:          rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding"),
			TypedBuilder: BuildClusterRoleBindingNode,
		},
		framework.ResourceRegistration{
			GVK:          corev1.SchemeGroupVersion.WithKind("ServiceAccount"),
			TypedBuilder: BuildServiceAccountNode,
		},
	)
	framework.RegisterDefaultCollections(
		framework.CollectionTarget{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "roles",
		},
		framework.CollectionTarget{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "rolebindings",
		},
		framework.CollectionTarget{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "clusterroles",
		},
		framework.CollectionTarget{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "clusterrolebindings",
		},
		framework.CollectionTarget{
			Version:  "v1",
			Resource: "serviceaccounts",
		},
	)
}
