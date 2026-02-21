package nodes

import "fmt"

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

	core := CoreEntry{
		Key:       "pods",
		Namespace: namespace,
		Cluster:   false,
		Data: map[string]any{
			"id":               BuildID("Pod", namespace, name),
			"kinds":            []string{"Pod"},
			"name":             name,
			"namespace":        namespace,
			"nodeName":         GetString(spec, "nodeName"),
			"hostPID":          GetBool(spec, "hostPID"),
			"serviceAccount":   GetString(spec, "serviceAccountName"),
			"labels_map":       labelsMap,
			"annotations_map":  annotationsMap,
			"containers":       privateContainers,
			"initContainers":   privateInitContainers,
			"volumes":          privateVolumes,
			"capabilitiesAdd":  capAdd,
			"capabilitiesDrop": capDrop,
			"seLinuxOptions":   seLinuxRaw,
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

func extractContainersDetail(containers []any, podSec map[string]any, podSeccomp map[string]any, podSeLinux map[string]any) []map[string]any {
	if len(containers) == 0 {
		return []map[string]any{}
	}

	results := make([]map[string]any, 0, len(containers))
	for _, item := range containers {
		container, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sec := GetMap(container, "securityContext")
		seccomp := GetMap(sec, "seccompProfile")
		seLinux := GetMap(sec, "seLinuxOptions")

		privileged := GetBool(sec, "privileged")
		readOnly := GetBool(sec, "readOnlyRootFilesystem")
		runAsUser := GetString(sec, "runAsUser")
		if runAsUser == "" {
			runAsUser = GetString(podSec, "runAsUser")
		}
		runAsGroup := GetString(sec, "runAsGroup")
		if runAsGroup == "" {
			runAsGroup = GetString(podSec, "runAsGroup")
		}
		runAsNonRoot := GetBool(sec, "runAsNonRoot")
		if runAsNonRoot == "" {
			runAsNonRoot = GetBool(podSec, "runAsNonRoot")
		}

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

		result := map[string]any{
			"name":                   GetString(container, "name"),
			"image":                  GetString(container, "image"),
			"privileged":             privileged,
			"runAsUser":              runAsUser,
			"runAsGroup":             runAsGroup,
			"runAsNonRoot":           runAsNonRoot,
			"readOnlyRootFilesystem": readOnly,
			"securityContext": map[string]any{
				"runAsUser":       runAsUser,
				"runAsGroup":      runAsGroup,
				"runAsNonRoot":    runAsNonRoot,
				"seccompProfile":  seccompType,
				"appArmorProfile": appArmor,
				"seLinuxOptions":  seLinuxRaw,
			},
			"envFrom":      extractEnvFrom(container),
			"hostPorts":    extractHostPorts(container),
			"volumeMounts": GetSlice(container, "volumeMounts"),
		}
		results = append(results, result)
	}

	return results
}

func extractInitContainersDetail(containers []any) []map[string]any {
	if len(containers) == 0 {
		return []map[string]any{}
	}
	results := make([]map[string]any, 0, len(containers))
	for _, item := range containers {
		container, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sec := GetMap(container, "securityContext")
		result := map[string]any{
			"name":       GetString(container, "name"),
			"image":      GetString(container, "image"),
			"privileged": GetBool(sec, "privileged"),
		}
		results = append(results, result)
	}
	return results
}

func extractEnvFrom(container map[string]any) []map[string]any {
	items := GetSlice(container, "envFrom")
	if len(items) == 0 {
		return []map[string]any{}
	}
	results := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		results = append(results, map[string]any{
			"secretRef":    entry["secretRef"],
			"configMapRef": entry["configMapRef"],
		})
	}
	return results
}

func extractHostPorts(container map[string]any) []map[string]any {
	items := GetSlice(container, "ports")
	if len(items) == 0 {
		return []map[string]any{}
	}
	results := make([]map[string]any, 0, len(items))
	for _, item := range items {
		port, ok := item.(map[string]any)
		if !ok {
			continue
		}
		hostPort := GetNumber(port, "hostPort")
		if hostPort == 0 {
			continue
		}
		results = append(results, map[string]any{
			"containerPort": GetNumber(port, "containerPort"),
			"hostPort":      hostPort,
			"hostIP":        GetString(port, "hostIP"),
			"protocol":      GetStringDefault(port, "protocol", "TCP"),
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

func summarizeContainers(containers []map[string]any, init bool) []string {
	if len(containers) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(containers))
	for _, container := range containers {
		name := GetString(container, "name")
		if init {
			name = fmt.Sprintf("init/%s", name)
		}
		items = append(items, fmt.Sprintf("%s: image=%v, privileged=%v, runAsUser=%v, runAsNonRoot=%v, readOnlyRootFilesystem=%v",
			name,
			container["image"],
			container["privileged"],
			container["runAsUser"],
			container["runAsNonRoot"],
			container["readOnlyRootFilesystem"],
		))
	}
	return items
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
