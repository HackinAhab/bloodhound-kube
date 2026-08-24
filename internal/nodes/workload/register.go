package workload

import (
	"bloodhound-kube/internal/nodes/framework"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func Register() {
	framework.RegisterResources(
		framework.ResourceRegistration{
			GVK:          corev1.SchemeGroupVersion.WithKind("Pod"),
			TypedBuilder: BuildPodNode,
		},
		framework.ResourceRegistration{
			GVK:          appsv1.SchemeGroupVersion.WithKind("Deployment"),
			TypedBuilder: BuildDeploymentNode,
		},
		framework.ResourceRegistration{
			GVK:          appsv1.SchemeGroupVersion.WithKind("DaemonSet"),
			TypedBuilder: BuildDaemonSetNode,
		},
		framework.ResourceRegistration{
			GVK:          appsv1.SchemeGroupVersion.WithKind("StatefulSet"),
			TypedBuilder: BuildStatefulSetNode,
		},
		framework.ResourceRegistration{
			GVK:          batchv1.SchemeGroupVersion.WithKind("Job"),
			TypedBuilder: BuildJobNode,
		},
		framework.ResourceRegistration{
			GVK:          batchv1.SchemeGroupVersion.WithKind("CronJob"),
			TypedBuilder: BuildCronJobNode,
		},
		framework.ResourceRegistration{
			GVK:          corev1.SchemeGroupVersion.WithKind("ConfigMap"),
			TypedBuilder: BuildConfigMapNode,
		},
		framework.ResourceRegistration{
			GVK:          corev1.SchemeGroupVersion.WithKind("Secret"),
			TypedBuilder: BuildSecretNode,
		},
	)
	framework.RegisterDefaultCollections(
		framework.CollectionTarget{
			Version:  "v1",
			Resource: "pods",
		},
		framework.CollectionTarget{
			Group:    "apps",
			Version:  "v1",
			Resource: "deployments",
		},
		framework.CollectionTarget{
			Group:    "apps",
			Version:  "v1",
			Resource: "daemonsets",
		},
		framework.CollectionTarget{
			Group:    "apps",
			Version:  "v1",
			Resource: "statefulsets",
		},
		framework.CollectionTarget{
			Group:    "batch",
			Version:  "v1",
			Resource: "jobs",
		},
		framework.CollectionTarget{
			Group:    "batch",
			Version:  "v1",
			Resource: "cronjobs",
		},
		framework.CollectionTarget{
			Version:  "v1",
			Resource: "configmaps",
		},
		framework.CollectionTarget{
			Version:  "v1",
			Resource: "secrets",
		},
	)
}
