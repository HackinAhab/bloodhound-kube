package workload

import (
	corev1 "k8s.io/api/core/v1"
)

func int64PointerWithFallback(
	containerSec *corev1.SecurityContext,
	podSec *corev1.PodSecurityContext,
	fromContainer func(*corev1.SecurityContext) *int64,
	fromPod func(*corev1.PodSecurityContext) *int64,
) *int64 {
	if containerSec != nil {
		if value := fromContainer(containerSec); value != nil {
			return value
		}
	}
	if podSec != nil {
		return fromPod(podSec)
	}
	return nil
}

func boolWithFallback(containerSec *corev1.SecurityContext, podSec *corev1.PodSecurityContext) bool {
	if containerSec != nil && containerSec.RunAsNonRoot != nil {
		return *containerSec.RunAsNonRoot
	}
	if podSec != nil && podSec.RunAsNonRoot != nil {
		return *podSec.RunAsNonRoot
	}
	return false
}

func inferPodRunAsUser(containers []corev1.Container, podSec *corev1.PodSecurityContext) any {
	if podSec != nil && podSec.RunAsUser != nil {
		return int(*podSec.RunAsUser)
	}

	if len(containers) == 0 {
		return "unset"
	}

	hasUnset := false
	anyZero := false
	var firstValue int64
	hasFirstValue := false
	allSame := true
	setCount := 0

	for _, container := range containers {
		value := effectiveContainerRunAsUser(container, podSec)
		if value == nil {
			hasUnset = true
			continue
		}
		setCount++
		if *value == 0 {
			anyZero = true
		}
		if !hasFirstValue {
			firstValue = *value
			hasFirstValue = true
			continue
		}
		if *value != firstValue {
			allSame = false
		}
	}

	if anyZero {
		return 0
	}
	if setCount == 0 || hasUnset || !allSame {
		return "unset"
	}
	return int(firstValue)
}

func inferPodRunAsNonRoot(containers []corev1.Container, podSec *corev1.PodSecurityContext) any {
	if podSec != nil && podSec.RunAsNonRoot != nil {
		return *podSec.RunAsNonRoot
	}

	if len(containers) == 0 {
		return "unset"
	}

	setCount := 0
	for _, container := range containers {
		value := effectiveContainerRunAsNonRoot(container, podSec)
		if value == nil {
			continue
		}
		setCount++
		if !*value {
			return false
		}
	}

	if setCount == len(containers) {
		return true
	}
	return "unset"
}

func effectiveContainerRunAsUser(container corev1.Container, podSec *corev1.PodSecurityContext) *int64 {
	if container.SecurityContext != nil && container.SecurityContext.RunAsUser != nil {
		return container.SecurityContext.RunAsUser
	}
	if podSec != nil {
		return podSec.RunAsUser
	}
	return nil
}

func effectiveContainerRunAsNonRoot(container corev1.Container, podSec *corev1.PodSecurityContext) *bool {
	if container.SecurityContext != nil && container.SecurityContext.RunAsNonRoot != nil {
		return container.SecurityContext.RunAsNonRoot
	}
	if podSec != nil {
		return podSec.RunAsNonRoot
	}
	return nil
}
