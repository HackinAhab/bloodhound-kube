package workload

import (
	corev1 "k8s.io/api/core/v1"
)

func extractVolumesDetail(volumes []corev1.Volume) []VolumeDetail {
	if len(volumes) == 0 {
		return []VolumeDetail{}
	}
	results := make([]VolumeDetail, 0, len(volumes))
	for _, volume := range volumes {
		results = append(results, VolumeDetail{
			Name:          volume.Name,
			Type:          volumeType(volume),
			SecretName:    volumeSecretName(volume),
			ConfigMapName: volumeConfigMapName(volume),
			PVCName:       volumePVCName(volume),
			HostPath:      volumeHostPath(volume),
		})
	}
	return results
}

func volumeType(volume corev1.Volume) string {
	if volume.Secret != nil {
		return "secret"
	}
	if volume.ConfigMap != nil {
		return "configmap"
	}
	if volume.PersistentVolumeClaim != nil {
		return "persistentVolumeClaim"
	}
	if volume.HostPath != nil {
		return "hostPath"
	}
	if volume.EmptyDir != nil {
		return "emptyDir"
	}
	if volume.Projected != nil {
		return "projected"
	}
	if volume.DownwardAPI != nil {
		return "downwardAPI"
	}
	return "other"
}

func volumeSecretName(volume corev1.Volume) string {
	if volume.Secret == nil {
		return ""
	}
	return volume.Secret.SecretName
}

func volumeConfigMapName(volume corev1.Volume) string {
	if volume.ConfigMap == nil {
		return ""
	}
	return volume.ConfigMap.Name
}

func volumePVCName(volume corev1.Volume) string {
	if volume.PersistentVolumeClaim == nil {
		return ""
	}
	return volume.PersistentVolumeClaim.ClaimName
}

func volumeHostPath(volume corev1.Volume) string {
	if volume.HostPath == nil {
		return ""
	}
	return volume.HostPath.Path
}
