package mounts

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

type HostMountReadEdgeRule struct{}

func (r HostMountReadEdgeRule) Name() string {
	return "lateral_movement_host_mount_read"
}

var edgePropertiesHostMountRead = map[string]any{
	"Description": "Pod has a host mount of sensitive directories, which can allow for lateral movement and potential information disclosure",
	"Reference":   "https://kubehound.io/reference/attacks/EXPLOIT_HOST_READ/",
}

func (r HostMountReadEdgeRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space != nil {
			edges = append(edges, podHostMountReadNamespaced(ctx, ns, space)...)
		}
	}
	return edges
}

func podHostMountReadNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for i := range space.Pods {
		pod := &space.Pods[i]
		if pod.NodeName == "" || pod.ID == "" {
			continue
		}
		node := ctx.Index.NodesByName[pod.NodeName]
		if node == nil || node.ID == "" {
			continue
		}
		mountPath, ok := podHostMountReadCheck(pod)
		if !ok {
			continue
		}
		description, _ := edgePropertiesHostMountRead["Description"].(string)
		reference := edgePropertiesHostMountRead["Reference"]
		edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "hostMountSensitive", map[string]any{
			"Description": description + " Mount path: " + mountPath,
			"Reference":   reference,
		}))
	}
	return edges
}

func podHostMountReadCheck(pod *nodes.Pod) (string, bool) {
	if pod == nil {
		return "", false
	}
	sensitivePaths := []string{"/etc", "/root", "/home", "/proc", "/var/lib/kubelet/pods"}
	volumeNames := map[string]struct{}{}
	for _, volume := range pod.Volumes {
		hostPath := volume.HostPath
		if hostPath == "" || !framework.HostPathMatchesAny(hostPath, sensitivePaths) {
			continue
		}
		if volume.Name != "" {
			volumeNames[volume.Name] = struct{}{}
		}
	}
	if len(volumeNames) == 0 {
		return "", false
	}
	for _, container := range pod.Containers {
		for _, mount := range container.VolumeMounts {
			if _, ok := volumeNames[mount.Name]; ok && mount.MountPath != "" {
				return mount.MountPath, true
			}
		}
	}
	return "", false
}

type HostMountKubeletEdgeRule struct{}

func (r HostMountKubeletEdgeRule) Name() string {
	return "lateral_movement_host_mount_kubelet"
}

var edgePropertiesHostMountKubelet = map[string]any{
	"Description": "Pod has a host mount containing a common kubelet directory, which may allow access to the node kubelet configuration and credentials. This check is left relatively broad to catch various kubelet-related host mounts, but may produce false positives and duplicates with LM_HOST_MOUNT_READ if common kubelet subdirectories are included in the host mount path.",
	"Reference":   "",
}

func (r HostMountKubeletEdgeRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space != nil {
			edges = append(edges, podHostMountKubeletNamespaced(ctx, ns, space)...)
		}
	}
	return edges
}

func podHostMountKubeletNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for i := range space.Pods {
		pod := &space.Pods[i]
		if pod.NodeName == "" || pod.ID == "" {
			continue
		}
		node := ctx.Index.NodesByName[pod.NodeName]
		if node == nil || node.ID == "" {
			continue
		}
		mountPath, ok := podHostMountKubeletCheck(pod)
		if !ok {
			continue
		}
		description, _ := edgePropertiesHostMountKubelet["Description"].(string)
		reference := edgePropertiesHostMountKubelet["Reference"]
		edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "mountedKubelet", map[string]any{
			"Description": description + " Mount path: " + mountPath,
			"Reference":   reference,
		}))
	}
	return edges
}

func podHostMountKubeletCheck(pod *nodes.Pod) (string, bool) {
	if pod == nil {
		return "", false
	}
	kubeletPaths := []string{"/var/lib/kubelet", "/etc/kubernetes"}
	volumeNames := map[string]struct{}{}
	for _, volume := range pod.Volumes {
		hostPath := volume.HostPath
		if hostPath == "" || !framework.HostPathMatchesAny(hostPath, kubeletPaths) {
			continue
		}
		if volume.Name != "" {
			volumeNames[volume.Name] = struct{}{}
		}
	}
	if len(volumeNames) == 0 {
		return "", false
	}
	for _, container := range pod.Containers {
		for _, mount := range container.VolumeMounts {
			if _, ok := volumeNames[mount.Name]; ok && mount.MountPath != "" {
				return mount.MountPath, true
			}
		}
	}
	return "", false
}

type PodMountServiceAccountEdgeRule struct{}

func (r PodMountServiceAccountEdgeRule) Name() string {
	return "lateral_movement_pod_mount_service_account"
}

var edgePropertiesPodMountServiceAccount = map[string]any{
	"Description": "Pod mounts a ServiceAccount token, which may have additional privileges.",
	"Reference":   "https://kubehound.io/reference/attacks/TOKEN_STEAL/",
}

func (r PodMountServiceAccountEdgeRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space != nil {
			edges = append(edges, podMountServiceAccountNamespaced(ctx, ns, space)...)
		}
	}
	return edges
}

func podMountServiceAccountNamespaced(ctx *framework.Context, namespace string, space *model.Namespace) []model.BloodHoundEdge {
	if ctx == nil || space == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	serviceAccounts := ctx.Index.ServiceAccountsByNamespace[namespace]
	for i := range space.Pods {
		pod := &space.Pods[i]
		if !podMountsServiceAccountToken(pod) {
			continue
		}
		saName := pod.ServiceAccount
		if saName == "" {
			saName = "default"
		}
		if saName == "" || serviceAccounts == nil {
			continue
		}
		serviceAccount := serviceAccounts[saName]
		if serviceAccount == nil || serviceAccount.ID == "" {
			continue
		}
		edges = append(edges, framework.CreateEdgeWithProperties(pod, serviceAccount, "mountedSA", edgePropertiesPodMountServiceAccount))
	}
	return edges
}

func podMountsServiceAccountToken(pod *nodes.Pod) bool {
	if pod == nil {
		return false
	}
	if pod.ServiceAccount != "default" {
		return pod.AutomountSAToken == nil || *pod.AutomountSAToken
	}
	return pod.AutomountSAToken != nil && *pod.AutomountSAToken
}
