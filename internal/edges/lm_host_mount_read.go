package edges

// TODO: Port LM_HOST_MOUNT_READ edge rule from rego.
import (
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

type lateralMovementHostMountReadEdgeRule struct{}

func init() {
	RegisterEdgeRule(lateralMovementHostMountReadEdgeRule{})
}

func (r lateralMovementHostMountReadEdgeRule) Name() string {
	return "lateral_movement_host_mount_read"
}

var edgePropertiesLateralMovementHostMountRead = map[string]any{
	"Description": "Pod has a host mount of sensitive directories, which can allow for lateral movement and potential information disclosure",
	"Reference":   "https://kubehound.io/reference/attacks/EXPLOIT_HOST_READ/",
}

func (r lateralMovementHostMountReadEdgeRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		edges = append(edges, podHostMountReadNamespaced(ctx, ns, space)...)
	}
	return edges
}

// Pod w/ host mount of sensitive Directories -> Nodes
func podHostMountReadNamespaced(ctx *EdgeContext, namespace string, space *model.Namespace) []model.BloodHoundEdge {
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
		description, _ := edgePropertiesLateralMovementHostMountRead["Description"].(string)
		reference := edgePropertiesLateralMovementHostMountRead["Reference"]
		props := map[string]any{
			"Description": description + " Mount path: " + mountPath,
			"Reference":   reference,
		}
		edges = append(edges, CreateEdgeWithProperties(pod, node, "LM_HOST_MOUNT_READ", props))
	}
	return edges
}

func podHostMountReadCheck(pod *nodes.Pod) (string, bool) {
	if pod == nil {
		return "", false
	}
	sensitivePaths := []string{
		"/etc",
		"/root",
		"/home",
		"/proc",
		"/var/lib/kubelet/pods",
	}
	volumeNames := map[string]struct{}{}
	for _, volume := range pod.Volumes {
		hostPath := volume.HostPath
		if hostPath == "" {
			continue
		}
		if !hostPathMatchesAny(hostPath, sensitivePaths) {
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
