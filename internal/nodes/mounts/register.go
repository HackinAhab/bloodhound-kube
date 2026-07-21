package mounts

import (
	"bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
)

func Register() {
	framework.RegisterTyped(corev1.SchemeGroupVersion.WithKind("PersistentVolume"), BuildPVNode)
	framework.RegisterTypedWithFetchMode(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"), BuildPVCNode, framework.FetchModeHintMetadata)
}
