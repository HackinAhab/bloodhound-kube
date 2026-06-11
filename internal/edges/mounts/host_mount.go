package mounts

import (
	"strings"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes/workload"
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

func podHostMountReadCheck(pod *workload.Pod) (string, bool) {
	if pod == nil {
		return "", false
	}
	sensitivePaths := []string{"/etc", "/root", "/home", "/proc", "/var/lib/kubelet/pods", "/var/run", "/sys", "/dev", "/run", "/usr"}
	volumeNames := map[string]struct{}{}
	for _, volume := range pod.Volumes {
		hostPath := volume.HostPath
		if hostPath == "" || strings.HasSuffix(hostPath, ".sock") || !framework.HostPathMatchesAny(hostPath, sensitivePaths) {
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
