package nodes

import "fmt"

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

type Pod struct {
	GraphNodeBase
	NodeName         string
	ServiceAccount   string
	Containers       []Container
	InitContainers   []Container
	Volumes          []map[string]any
	CapabilitiesAdd  []string
	CapabilitiesDrop []string
	SeLinuxOptions   map[string]any
	HostPID          bool
}

func init() {
	Register("Pod", BuildPodNode)
}

func BuildPodNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}

	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	securityContext := GetMap(spec, "securityContext")
	seccompProfile := GetMap(securityContext, "seccompProfile")
	seLinuxRaw := GetMap(securityContext, "seLinuxOptions")

	capAdd, capDrop := extractCapabilities(spec)
	containerImages := extractContainerImages(GetSlice(spec, "containers"))
	initContainerImages := extractContainerImages(GetSlice(spec, "initContainers"))

	privateContainers := extractContainersDetail(GetSlice(spec, "containers"), securityContext, seccompProfile, seLinuxRaw)
	privateInitContainers := extractInitContainersDetail(GetSlice(spec, "initContainers"))
	privateVolumes := extractVolumesDetail(GetSlice(spec, "volumes"))

	containerSummaries := summarizeContainers(privateContainers, false)
	initContainerSummaries := summarizeContainers(privateInitContainers, true)
	volumeSummaries := summarizeVolumes(privateVolumes)

	properties := map[string]any{
		"name":                      name,
		"namespace":                 namespace,
		"labels":                    MapToSortedList(labelsMap),
		"annotations":               MapToSortedList(annotationsMap),
		"securityContextConstraint": GetString(annotationsMap, "openshift.io/scc"),
		"nodeName":                  GetString(spec, "nodeName"),
		"serviceAccount":            GetString(spec, "serviceAccountName"),
		"containers":                containerSummaries,
		"initContainers":            initContainerSummaries,
		"containerImages":           containerImages,
		"initContainerImages":       initContainerImages,
		"capabilitiesAdd":           capAdd,
		"capabilitiesDrop":          capDrop,
		"hostNetwork":               GetBool(spec, "hostNetwork"),
		"hostPid":                   GetBool(spec, "hostPID"),
		"hostIpc":                   GetBool(spec, "hostIPC"),
		"runAsUser":                 GetNumber(securityContext, "runAsUser"),
		"runAsGroup":                GetNumber(securityContext, "runAsGroup"),
		"runAsNonRoot":              GetBool(securityContext, "runAsNonRoot"),
		"fsGroup":                   GetNumber(securityContext, "fsGroup"),
		"supplementalGroups":        GetSlice(securityContext, "supplementalGroups"),
		"seccompProfile":            GetString(seccompProfile, "type"),
		"appArmorProfile":           AppArmorProfileValue(securityContext),
		"seLinuxOptions":            SeLinuxSummary(seLinuxRaw),
		"volumes":                   volumeSummaries,
	}

	hostPID := GetBoolValue(spec, "hostPID")
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
			NodeName:         GetString(spec, "nodeName"),
			ServiceAccount:   GetString(spec, "serviceAccountName"),
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

func extractContainerImages(containers []any) []string {
	if len(containers) == 0 {
		return []string{}
	}
	images := make([]string, 0, len(containers))
	for _, item := range containers {
		container, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if image, ok := container["image"].(string); ok {
			images = append(images, image)
		}
	}
	return images
}

func extractCapabilities(spec map[string]any) ([]string, []string) {
	containers := GetSlice(spec, "containers")
	addSet := map[string]struct{}{}
	dropSet := map[string]struct{}{}

	for _, item := range containers {
		container, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sec := GetMap(container, "securityContext")
		caps := GetMap(sec, "capabilities")
		for _, cap := range GetSlice(caps, "add") {
			if s, ok := cap.(string); ok {
				addSet[s] = struct{}{}
			}
		}
		for _, cap := range GetSlice(caps, "drop") {
			if s, ok := cap.(string); ok {
				dropSet[s] = struct{}{}
			}
		}
	}

	add := setToSortedList(addSet)
	drop := setToSortedList(dropSet)
	return add, drop
}

func setToSortedList(set map[string]struct{}) []string {
	return SortedSetKeys(set)
}

func extractContainersDetail(containers []any, podSec map[string]any, podSeccomp map[string]any, podSeLinux map[string]any) []Container {
	if len(containers) == 0 {
		return []Container{}
	}

	results := make([]Container, 0, len(containers))
	for _, item := range containers {
		container, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sec := GetMap(container, "securityContext")
		seccomp := GetMap(sec, "seccompProfile")
		seLinux := GetMap(sec, "seLinuxOptions")

		privileged := GetBoolValue(sec, "privileged")
		readOnly := GetBoolValue(sec, "readOnlyRootFilesystem")
		runAsUser := int64PointerWithFallback(sec, podSec, "runAsUser")
		runAsGroup := int64PointerWithFallback(sec, podSec, "runAsGroup")
		runAsNonRoot := boolWithFallback(sec, podSec, "runAsNonRoot")

		seccompType := GetString(seccomp, "type")
		if seccompType == "" {
			seccompType = GetString(podSeccomp, "type")
		}
		appArmor := AppArmorProfileValue(sec)
		if appArmor == "" {
			appArmor = AppArmorProfileValue(podSec)
		}

		seLinuxRaw := seLinux
		if len(seLinuxRaw) == 0 {
			seLinuxRaw = podSeLinux
		}

		result := Container{
			Name:                   GetString(container, "name"),
			Image:                  GetString(container, "image"),
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
				Raw:                    sec,
			},
			EnvFrom:      extractEnvFrom(container),
			HostPorts:    extractHostPorts(container),
			VolumeMounts: extractVolumeMounts(container),
			Raw:          container,
		}
		results = append(results, result)
	}

	return results
}

func extractInitContainersDetail(containers []any) []Container {
	if len(containers) == 0 {
		return []Container{}
	}
	results := make([]Container, 0, len(containers))
	for _, item := range containers {
		container, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sec := GetMap(container, "securityContext")
		result := Container{
			Name:       GetString(container, "name"),
			Image:      GetString(container, "image"),
			Privileged: GetBoolValue(sec, "privileged"),
			Raw:        container,
		}
		results = append(results, result)
	}
	return results
}

func extractEnvFrom(container map[string]any) []EnvFromSource {
	items := GetSlice(container, "envFrom")
	if len(items) == 0 {
		return []EnvFromSource{}
	}
	results := make([]EnvFromSource, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result := EnvFromSource{
			Raw: entry,
		}
		if ref, ok := entry["secretRef"].(map[string]any); ok {
			result.SecretRef = &NamedObjectRef{Name: GetString(ref, "name")}
		}
		if ref, ok := entry["configMapRef"].(map[string]any); ok {
			result.ConfigMapRef = &NamedObjectRef{Name: GetString(ref, "name")}
		}
		results = append(results, result)
	}
	return results
}

func extractHostPorts(container map[string]any) []HostPort {
	items := GetSlice(container, "ports")
	if len(items) == 0 {
		return []HostPort{}
	}
	results := make([]HostPort, 0, len(items))
	for _, item := range items {
		port, ok := item.(map[string]any)
		if !ok {
			continue
		}
		hostPort := GetNumber(port, "hostPort")
		if hostPort == 0 {
			continue
		}
		results = append(results, HostPort{
			ContainerPort: GetNumber(port, "containerPort"),
			HostPort:      hostPort,
			HostIP:        GetString(port, "hostIP"),
			Protocol:      GetStringDefault(port, "protocol", "TCP"),
			Raw:           port,
		})
	}
	return results
}

func extractVolumeMounts(container map[string]any) []VolumeMount {
	items := GetSlice(container, "volumeMounts")
	if len(items) == 0 {
		return []VolumeMount{}
	}
	results := make([]VolumeMount, 0, len(items))
	for _, item := range items {
		mount, ok := item.(map[string]any)
		if !ok {
			continue
		}
		results = append(results, VolumeMount{
			Name:      GetString(mount, "name"),
			MountPath: GetString(mount, "mountPath"),
			ReadOnly:  GetBoolValue(mount, "readOnly"),
			Raw:       mount,
		})
	}
	return results
}

func extractVolumesDetail(volumes []any) []map[string]any {
	if len(volumes) == 0 {
		return []map[string]any{}
	}
	results := make([]map[string]any, 0, len(volumes))
	for _, item := range volumes {
		volume, ok := item.(map[string]any)
		if !ok {
			continue
		}
		results = append(results, map[string]any{
			"name":          GetString(volume, "name"),
			"type":          volumeType(volume),
			"secretName":    GetString(GetMap(volume, "secret"), "secretName"),
			"configMapName": GetString(GetMap(volume, "configMap"), "name"),
			"pvcName":       GetString(GetMap(volume, "persistentVolumeClaim"), "claimName"),
			"hostPath":      GetString(GetMap(volume, "hostPath"), "path"),
		})
	}
	return results
}

func volumeType(volume map[string]any) string {
	if _, ok := volume["secret"]; ok {
		return "secret"
	}
	if _, ok := volume["configMap"]; ok {
		return "configmap"
	}
	if _, ok := volume["persistentVolumeClaim"]; ok {
		return "persistentVolumeClaim"
	}
	if _, ok := volume["hostPath"]; ok {
		return "hostPath"
	}
	if _, ok := volume["emptyDir"]; ok {
		return "emptyDir"
	}
	if _, ok := volume["projected"]; ok {
		return "projected"
	}
	if _, ok := volume["downwardAPI"]; ok {
		return "downwardAPI"
	}
	return "other"
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

func int64PointerWithFallback(primary, fallback map[string]any, key string) *int64 {
	if _, ok := primary[key]; ok {
		if value := GetInt64Pointer(primary, key); value != nil {
			return value
		}
	}
	return GetInt64Pointer(fallback, key)
}

func boolWithFallback(primary, fallback map[string]any, key string) bool {
	if _, ok := primary[key]; ok {
		return GetBoolValue(primary, key)
	}
	return GetBoolValue(fallback, key)
}

func int64PointerValue(value *int64) any {
	if value == nil {
		return ""
	}
	return *value
}

func summarizeVolumes(volumes []map[string]any) []string {
	if len(volumes) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(volumes))
	for _, volume := range volumes {
		items = append(items, fmt.Sprintf("%s: type=%v, secret=%v, configMap=%v, pvc=%v, hostPath=%v",
			volume["name"],
			volume["type"],
			volume["secretName"],
			volume["configMapName"],
			volume["pvcName"],
			volume["hostPath"],
		))
	}
	return items
}
