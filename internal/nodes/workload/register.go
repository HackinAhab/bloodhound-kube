package workload

import (
	"bloodhound-kube/internal/nodes/framework"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func Register(reg *framework.Registry) {
	reg.RegisterTyped(corev1.SchemeGroupVersion.WithKind("Pod"), BuildPodNode)
	reg.RegisterTyped(appsv1.SchemeGroupVersion.WithKind("Deployment"), BuildDeploymentNode)
	reg.RegisterTyped(appsv1.SchemeGroupVersion.WithKind("DaemonSet"), BuildDaemonSetNode)
	reg.RegisterTyped(appsv1.SchemeGroupVersion.WithKind("StatefulSet"), BuildStatefulSetNode)
	reg.RegisterTyped(batchv1.SchemeGroupVersion.WithKind("Job"), BuildJobNode)
	reg.RegisterTyped(batchv1.SchemeGroupVersion.WithKind("CronJob"), BuildCronJobNode)
	reg.RegisterTyped(corev1.SchemeGroupVersion.WithKind("ConfigMap"), BuildConfigMapNode)
	reg.RegisterTyped(corev1.SchemeGroupVersion.WithKind("Secret"), BuildSecretNode)
}
