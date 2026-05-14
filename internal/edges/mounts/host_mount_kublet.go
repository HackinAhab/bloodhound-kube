package mounts

import (
	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes/workload"
)

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

func podHostMountKubeletCheck(pod *workload.Pod) (string, bool) {
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
