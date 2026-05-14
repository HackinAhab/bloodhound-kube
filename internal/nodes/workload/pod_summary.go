package workload

import (
	"fmt"
	"strings"
)

func summarizeContainers(containers []Container, init bool) []string {
	if len(containers) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(containers))
	for _, container := range containers {
		name := container.Name
		if init {
			name = fmt.Sprintf("init/%s", name)
		}
		items = append(items, fmt.Sprintf("%s: image=%v, privileged=%v, runAsUser=%v, runAsNonRoot=%v, readOnlyRootFilesystem=%v",
			name,
			container.Image,
			container.Privileged,
			int64PointerValue(container.RunAsUser),
			container.RunAsNonRoot,
			container.ReadOnlyRootFilesystem,
		))
	}
	return items
}

func int64PointerValue(value *int64) any {
	if value == nil {
		return ""
	}
	return *value
}

func summarizeVolumes(volumes []VolumeDetail) []string {
	if len(volumes) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(volumes))
	for _, volume := range volumes {
		parts := []string{fmt.Sprintf("type=%s", volume.Type)}
		switch volume.Type {
		case "secret":
			if volume.SecretName != "" {
				parts = append(parts, fmt.Sprintf("secret=%s", volume.SecretName))
			}
		case "configmap":
			if volume.ConfigMapName != "" {
				parts = append(parts, fmt.Sprintf("configMap=%s", volume.ConfigMapName))
			}
		case "persistentVolumeClaim":
			if volume.PVCName != "" {
				parts = append(parts, fmt.Sprintf("pvc=%s", volume.PVCName))
			}
		case "hostPath":
			if volume.HostPath != "" {
				parts = append(parts, fmt.Sprintf("hostPath=%s", volume.HostPath))
			}
		}

		items = append(items, fmt.Sprintf("%s: %s", volume.Name, strings.Join(parts, ", ")))
	}
	return items
}
