package platform

import (
	"bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Register(reg *framework.Registry) {
	reg.Register("Namespace", BuildNamespaceNode)
	reg.RegisterTypedFromMapWithFetchMode(corev1.SchemeGroupVersion.WithKind("Namespace"), BuildNamespaceNode, framework.FetchModeHintMetadata)
	reg.RegisterTypedWithFetchMode(schema.GroupVersion{Group: "", Version: "v1"}.WithKind("Node"), BuildNodeNode, framework.FetchModeHintMetadata)
}
