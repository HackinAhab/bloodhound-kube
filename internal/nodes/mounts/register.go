package mounts

import (
	"bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
)

func Register() {
	framework.RegisterResources(
		framework.ResourceRegistration{
			GVK:          corev1.SchemeGroupVersion.WithKind("PersistentVolume"),
			TypedBuilder: BuildPVNode,
		},
		framework.ResourceRegistration{
			GVK:          corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"),
			TypedBuilder: BuildPVCNode,
			FetchMode:    framework.FetchModeHintMetadata,
		},
	)
	framework.RegisterDefaultCollections(
		framework.CollectionTarget{
			Version:  "v1",
			Resource: "persistentvolumes",
		},
		framework.CollectionTarget{
			Version:  "v1",
			Resource: "persistentvolumeclaims",
		},
	)
}
