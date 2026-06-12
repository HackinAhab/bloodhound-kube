package mounts

import (
	"bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
)

func Register(reg *framework.Registry) {
	reg.RegisterTyped(corev1.SchemeGroupVersion.WithKind("PersistentVolume"), BuildPVNode)
	reg.RegisterTypedWithFetchMode(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"), BuildPVCNode, framework.FetchModeHintMetadata)
}
