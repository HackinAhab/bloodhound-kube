package platform

import (
	"bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Register() {
	framework.RegisterResources(
		framework.ResourceRegistration{
			GVK:        corev1.SchemeGroupVersion.WithKind("Namespace"),
			MapBuilder: BuildNamespaceNode,
			FetchMode:  framework.FetchModeHintMetadata,
		},
		framework.ResourceRegistration{
			GVK:          schema.GroupVersion{Group: "", Version: "v1"}.WithKind("Node"),
			TypedBuilder: BuildNodeNode,
			FetchMode:    framework.FetchModeHintMetadata,
		},
	)
	framework.RegisterDefaultCollections(framework.CollectionTarget{
		Version:  "v1",
		Resource: "nodes",
	})
}
