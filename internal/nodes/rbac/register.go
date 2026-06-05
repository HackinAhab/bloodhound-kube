package rbac

import (
	"bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func Register(reg *framework.Registry) {
	reg.RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("Role"), BuildRoleNode)
	reg.RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("RoleBinding"), BuildRoleBindingNode)
	reg.RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("ClusterRole"), BuildClusterRoleNode)
	reg.RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding"), BuildClusterRoleBindingNode)
	reg.RegisterTyped(corev1.SchemeGroupVersion.WithKind("ServiceAccount"), BuildServiceAccountNode)
}
