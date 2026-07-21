package workload

import (
	"bloodhound-kube/internal/nodes/framework"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func Register() {
	framework.RegisterTyped(corev1.SchemeGroupVersion.WithKind("Pod"), BuildPodNode)
	framework.RegisterTyped(appsv1.SchemeGroupVersion.WithKind("Deployment"), BuildDeploymentNode)
	framework.RegisterTyped(appsv1.SchemeGroupVersion.WithKind("DaemonSet"), BuildDaemonSetNode)
	framework.RegisterTyped(appsv1.SchemeGroupVersion.WithKind("StatefulSet"), BuildStatefulSetNode)
	framework.RegisterTyped(batchv1.SchemeGroupVersion.WithKind("Job"), BuildJobNode)
	framework.RegisterTyped(batchv1.SchemeGroupVersion.WithKind("CronJob"), BuildCronJobNode)
	framework.RegisterTyped(corev1.SchemeGroupVersion.WithKind("ConfigMap"), BuildConfigMapNode)
	framework.RegisterTyped(corev1.SchemeGroupVersion.WithKind("Secret"), BuildSecretNode)
}
