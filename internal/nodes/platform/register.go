package platform

import (
	"bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Register() {
	framework.RegisterKind("Namespace", BuildNamespaceNode)
	framework.RegisterTypedFromMapWithFetchMode(corev1.SchemeGroupVersion.WithKind("Namespace"), BuildNamespaceNode, framework.FetchModeHintMetadata)
	framework.RegisterTypedWithFetchMode(schema.GroupVersion{Group: "", Version: "v1"}.WithKind("Node"), BuildNodeNode, framework.FetchModeHintMetadata)
}
