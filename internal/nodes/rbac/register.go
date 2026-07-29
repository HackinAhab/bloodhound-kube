package rbac

import (
	"bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func Register() {
	framework.RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("Role"), BuildRoleNode)
	framework.RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("RoleBinding"), BuildRoleBindingNode)
	framework.RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("ClusterRole"), BuildClusterRoleNode)
	framework.RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding"), BuildClusterRoleBindingNode)
	framework.RegisterTyped(corev1.SchemeGroupVersion.WithKind("ServiceAccount"), BuildServiceAccountNode)
}
