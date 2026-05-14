package workload

import (
	. "bloodhound-kube/internal/nodes/framework"

	corev1 "k8s.io/api/core/v1"
)

func extractContainerImages(containers []corev1.Container) []string {
	if len(containers) == 0 {
		return []string{}
	}
	images := make([]string, 0, len(containers))
	for _, container := range containers {
		if container.Image != "" {
			images = append(images, container.Image)
		}
	}
	return images
}

func extractCapabilitiesFromContainers(containers []corev1.Container) ([]string, []string) {
	addSet := map[string]struct{}{}
	dropSet := map[string]struct{}{}
	for _, container := range containers {
		sec := container.SecurityContext
		if sec == nil || sec.Capabilities == nil {
			continue
		}
		for _, cap := range sec.Capabilities.Add {
			addSet[string(cap)] = struct{}{}
		}
		for _, cap := range sec.Capabilities.Drop {
			dropSet[string(cap)] = struct{}{}
		}
	}
	add := SortedSetKeys(addSet)
	drop := SortedSetKeys(dropSet)
	return add, drop
}

func extractContainersDetail(containers []corev1.Container, podSec *corev1.PodSecurityContext) []Container {
	if len(containers) == 0 {
		return []Container{}
	}

	results := make([]Container, 0, len(containers))
	for _, container := range containers {
		sec := container.SecurityContext
		var seccompType string
		var appArmor string
		var seLinuxRaw map[string]any

		if sec != nil && sec.SeccompProfile != nil {
			seccompType = string(sec.SeccompProfile.Type)
		}
		if seccompType == "" && podSec != nil && podSec.SeccompProfile != nil {
			seccompType = string(podSec.SeccompProfile.Type)
		}

		if sec != nil {
			appArmor = AppArmorProfileValue(sec.AppArmorProfile)
		}
		if appArmor == "" && podSec != nil {
			appArmor = AppArmorProfileValue(podSec.AppArmorProfile)
		}

		seLinuxRaw = map[string]any{}
		if sec != nil && sec.SELinuxOptions != nil {
			seLinuxRaw = SeLinuxOptionsToMap(sec.SELinuxOptions)
		} else if podSec != nil && podSec.SELinuxOptions != nil {
			seLinuxRaw = SeLinuxOptionsToMap(podSec.SELinuxOptions)
		}

		privileged := false
		readOnly := false
		if sec != nil {
			if sec.Privileged != nil {
				privileged = *sec.Privileged
			}
			if sec.ReadOnlyRootFilesystem != nil {
				readOnly = *sec.ReadOnlyRootFilesystem
			}
		}

		runAsUser := int64PointerWithFallback(sec, podSec, func(ctx *corev1.SecurityContext) *int64 {
			if ctx == nil {
				return nil
			}
			return ctx.RunAsUser
		}, func(ctx *corev1.PodSecurityContext) *int64 {
			if ctx == nil {
				return nil
			}
			return ctx.RunAsUser
		})
		runAsGroup := int64PointerWithFallback(sec, podSec, func(ctx *corev1.SecurityContext) *int64 {
			if ctx == nil {
				return nil
			}
			return ctx.RunAsGroup
		}, func(ctx *corev1.PodSecurityContext) *int64 {
			if ctx == nil {
				return nil
			}
			return ctx.RunAsGroup
		})
		runAsNonRoot := boolWithFallback(sec, podSec)

		result := Container{
			Name:                   container.Name,
			Image:                  container.Image,
			Privileged:             privileged,
			RunAsUser:              runAsUser,
			RunAsGroup:             runAsGroup,
			RunAsNonRoot:           runAsNonRoot,
			ReadOnlyRootFilesystem: readOnly,
			SecurityContext: ContainerSecurityContext{
				RunAsUser:              runAsUser,
				RunAsGroup:             runAsGroup,
				RunAsNonRoot:           runAsNonRoot,
				SeccompProfile:         seccompType,
				AppArmorProfile:        appArmor,
				SeLinuxOptions:         seLinuxRaw,
				ReadOnlyRootFilesystem: readOnly,
				Privileged:             privileged,
				Raw:                    nil,
			},
			EnvFrom:      extractEnvFrom(container.EnvFrom),
			HostPorts:    extractHostPorts(container.Ports),
			VolumeMounts: extractVolumeMounts(container.VolumeMounts),
			Raw:          nil,
		}
		results = append(results, result)
	}

	return results
}

func extractInitContainersDetail(containers []corev1.Container) []Container {
	if len(containers) == 0 {
		return []Container{}
	}
	results := make([]Container, 0, len(containers))
	for _, container := range containers {
		privileged := false
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil {
			privileged = *container.SecurityContext.Privileged
		}
		result := Container{
			Name:       container.Name,
			Image:      container.Image,
			Privileged: privileged,
			Raw:        nil,
		}
		results = append(results, result)
	}
	return results
}

func extractEnvFrom(items []corev1.EnvFromSource) []EnvFromSource {
	if len(items) == 0 {
		return []EnvFromSource{}
	}
	results := make([]EnvFromSource, 0, len(items))
	for _, entry := range items {
		result := EnvFromSource{}
		if entry.SecretRef != nil {
			result.SecretRef = &NamedObjectRef{Name: entry.SecretRef.Name}
		}
		if entry.ConfigMapRef != nil {
			result.ConfigMapRef = &NamedObjectRef{Name: entry.ConfigMapRef.Name}
		}
		results = append(results, result)
	}
	return results
}

func extractHostPorts(ports []corev1.ContainerPort) []HostPort {
	if len(ports) == 0 {
		return []HostPort{}
	}
	results := make([]HostPort, 0, len(ports))
	for _, port := range ports {
		if port.HostPort == 0 {
			continue
		}
		protocol := string(port.Protocol)
		if protocol == "" {
			protocol = "TCP"
		}
		results = append(results, HostPort{
			ContainerPort: int(port.ContainerPort),
			HostPort:      int(port.HostPort),
			HostIP:        port.HostIP,
			Protocol:      protocol,
			Raw:           nil,
		})
	}
	return results
}

func extractVolumeMounts(mounts []corev1.VolumeMount) []VolumeMount {
	if len(mounts) == 0 {
		return []VolumeMount{}
	}
	results := make([]VolumeMount, 0, len(mounts))
	for _, mount := range mounts {
		results = append(results, VolumeMount{
			Name:      mount.Name,
			MountPath: mount.MountPath,
			ReadOnly:  mount.ReadOnly,
			Raw:       nil,
		})
	}
	return results
}
