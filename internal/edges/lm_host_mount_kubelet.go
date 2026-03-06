package edges

import (
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

type lateralMovementHostMountKubeletEdgeRule struct{}

func init() {
	RegisterEdgeRule(lateralMovementHostMountKubeletEdgeRule{})
}

func (r lateralMovementHostMountKubeletEdgeRule) Name() string {
	return "lateral_movement_host_mount_kubelet"
}

var edgePropertiesLateralMovementHostMountKubelet = map[string]any{
	"Description": "Pod has a host mount containing a common kubelet directory, which may allow access to the node kubelet configuration and credentials. This check is left relatively broad to catch various kubelet-related host mounts, but may produce false positives and duplicates with LM_HOST_MOUNT_READ if common kubelet subdirectories are included in the host mount path.",
	"Reference":   "",
}

func (r lateralMovementHostMountKubeletEdgeRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, podHostMountKubeletNamespaced(ctx, ns, space)...)
	}
	return edges
}

// Pod w/ host mount of kubelet -> Nodes
func podHostMountKubeletNamespaced(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
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
		description, _ := edgePropertiesLateralMovementHostMountKubelet["Description"].(string)
		reference := edgePropertiesLateralMovementHostMountKubelet["Reference"]
		props := map[string]any{
			"Description": description + " Mount path: " + mountPath,
			"Reference":   reference,
		}
		edges = append(edges, CreateEdgeWithProperties(pod, node, "LM_HOST_MOUNT_KUBELET", props))
	}
	return edges
}

func podHostMountKubeletCheck(pod *nodes.Pod) (string, bool) {
	if pod == nil {
		return "", false
	}
	kubeletPaths := []string{
		// TODO: Consider narrowing the scope of this check to specific kubelet filenames
		"/var/lib/kubelet",
		// "/var/lib/kubelet/config.yaml",
		"/etc/kubernetes",
		// "/etc/kubernetes/kubelet.conf",
	}
	volumeNames := map[string]struct{}{}
	for _, volume := range pod.Volumes {
		hostPath := volume.HostPath
		if hostPath == "" {
			continue
		}
		if !hostPathMatchesAny(hostPath, kubeletPaths) {
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
			if _, ok := volumeNames[mount.Name]; !ok {
				continue
			}
			if mount.MountPath == "" {
				continue
			}
			return mount.MountPath, true
		}
	}
	return "", false
}
