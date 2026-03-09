package nodes

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type VolumeMount struct {
	Name      string
	MountPath string
	ReadOnly  bool
	Raw       map[string]any
}

type HostPort struct {
	ContainerPort int
	HostPort      int
	HostIP        string
	Protocol      string
	Raw           map[string]any
}

type NamedObjectRef struct {
	Name string
}

type EnvFromSource struct {
	SecretRef    *NamedObjectRef
	ConfigMapRef *NamedObjectRef
	Raw          map[string]any
}

type ContainerSecurityContext struct {
	RunAsUser              *int64
	RunAsGroup             *int64
	RunAsNonRoot           bool
	SeccompProfile         string
	AppArmorProfile        string
	SeLinuxOptions         map[string]any
	ReadOnlyRootFilesystem bool
	Privileged             bool
	Raw                    map[string]any
}

type Container struct {
	Name                   string
	Image                  string
	Privileged             bool
	RunAsUser              *int64
	RunAsGroup             *int64
	RunAsNonRoot           bool
	ReadOnlyRootFilesystem bool
	SecurityContext        ContainerSecurityContext
	EnvFrom                []EnvFromSource
	HostPorts              []HostPort
	VolumeMounts           []VolumeMount
	Raw                    map[string]any
}

type VolumeDetail struct {
	Name          string
	Type          string
	SecretName    string
	ConfigMapName string
	PVCName       string
	HostPath      string
}

type Pod struct {
	GraphNodeBase
	NodeName         string
	ServiceAccount   string
	AutomountSAToken *bool
	ShareProcNs      *bool
	Containers       []Container
	InitContainers   []Container
	Volumes          []VolumeDetail
	CapabilitiesAdd  []string
	CapabilitiesDrop []string
	SeLinuxOptions   map[string]any
	HostPID          bool
}

func BuildPodNode(obj runtime.Object) (BuildResult, bool) {
	pod, ok := obj.(*corev1.Pod)
	if !ok || pod == nil {
		return BuildResult{}, false
	}
	name := pod.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := pod.Namespace
	labelsMap := StringMapToAnyMap(pod.Labels)
	annotationsMap := StringMapToAnyMap(pod.Annotations)

	var podSec *corev1.PodSecurityContext
	if pod.Spec.SecurityContext != nil {
		podSec = pod.Spec.SecurityContext
	}
	seccompProfile := ""
	seLinuxRaw := map[string]any{}
	if podSec != nil {
		if podSec.SeccompProfile != nil {
			seccompProfile = string(podSec.SeccompProfile.Type)
		}
		seLinuxRaw = SeLinuxOptionsToMap(podSec.SELinuxOptions)
	}

	capAdd, capDrop := extractCapabilitiesFromContainers(pod.Spec.Containers)
	containerImages := extractContainerImages(pod.Spec.Containers)
	initContainerImages := extractContainerImages(pod.Spec.InitContainers)

	privateContainers := extractContainersDetail(pod.Spec.Containers, podSec)
	privateInitContainers := extractInitContainersDetail(pod.Spec.InitContainers)
	privateVolumes := extractVolumesDetail(pod.Spec.Volumes)

	containerSummaries := summarizeContainers(privateContainers, false)
	initContainerSummaries := summarizeContainers(privateInitContainers, true)
	volumeSummaries := summarizeVolumes(privateVolumes)

	runAsUser := 0
	runAsGroup := 0
	fsGroup := 0
	var supplementalGroups []int64
	var runAsNonRoot any = false
	if podSec != nil {
		if podSec.RunAsUser != nil {
			runAsUser = int(*podSec.RunAsUser)
		}
		if podSec.RunAsGroup != nil {
			runAsGroup = int(*podSec.RunAsGroup)
		}
		if podSec.FSGroup != nil {
			fsGroup = int(*podSec.FSGroup)
		}
		if podSec.RunAsNonRoot != nil {
			runAsNonRoot = *podSec.RunAsNonRoot
		}
		supplementalGroups = podSec.SupplementalGroups
	}

	appArmorPod := ""
	if podSec != nil {
		appArmorPod = AppArmorProfileValue(podSec.AppArmorProfile)
	}

	shareProcessNamespace := false
	if pod.Spec.ShareProcessNamespace != nil {
		shareProcessNamespace = *pod.Spec.ShareProcessNamespace
	}
	properties := map[string]any{
		"name":                      name,
		"namespace":                 namespace,
		"labels":                    MapToSortedList(labelsMap),
		"annotations":               MapToSortedList(annotationsMap),
		"securityContextConstraint": annotationsMap["openshift.io/scc"],
		"nodeName":                  pod.Spec.NodeName,
		"serviceAccount":            pod.Spec.ServiceAccountName,
		"containers":                containerSummaries,
		"initContainers":            initContainerSummaries,
		"containerImages":           containerImages,
		"initContainerImages":       initContainerImages,
		"capabilitiesAdd":           capAdd,
		"capabilitiesDrop":          capDrop,
		"hostNetwork":               pod.Spec.HostNetwork,
		"hostPid":                   pod.Spec.HostPID,
		"hostIpc":                   pod.Spec.HostIPC,
		"shareProcessNamespace":     shareProcessNamespace,
		"runAsUser":                 runAsUser,
		"runAsGroup":                runAsGroup,
		"runAsNonRoot":              runAsNonRoot,
		"fsGroup":                   fsGroup,
		"supplementalGroups":        supplementalGroups,
		"seccompProfile":            seccompProfile,
		"appArmorProfile":           appArmorPod,
		"seLinuxOptions":            SeLinuxSummary(seLinuxRaw),
		"volumes":                   volumeSummaries,
	}

	hostPID := pod.Spec.HostPID
	automountSAToken := pod.Spec.AutomountServiceAccountToken
	shareProcNs := pod.Spec.ShareProcessNamespace
	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Pod{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("Pod", namespace, name),
				Kinds:          []string{"Pod"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			NodeName:         pod.Spec.NodeName,
			ServiceAccount:   pod.Spec.ServiceAccountName,
			AutomountSAToken: automountSAToken,
			ShareProcNs:      shareProcNs,
			Containers:       privateContainers,
			InitContainers:   privateInitContainers,
			Volumes:          privateVolumes,
			CapabilitiesAdd:  capAdd,
			CapabilitiesDrop: capDrop,
			SeLinuxOptions:   seLinuxRaw,
			HostPID:          hostPID,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Pod", namespace, name),
			Kinds:      []string{"Pod"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

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
		items = append(items, fmt.Sprintf("%s: type=%v, secret=%v, configMap=%v, pvc=%v, hostPath=%v",
			volume.Name,
			volume.Type,
			volume.SecretName,
			volume.ConfigMapName,
			volume.PVCName,
			volume.HostPath,
		))
	}
	return items
}

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
