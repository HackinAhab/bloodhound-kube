package edges

import (
	"strings"

	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

type attackEdgesRule struct{}

func (r attackEdgesRule) Name() string {
	return "attacks"
}

func (r attackEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for _, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		for i := range space.Pods {
			pod := &space.Pods[i]
			if pod.NodeName == "" || pod.ID == "" {
				continue
			}
			node := ctx.Index.NodesByName[pod.NodeName]
			if node == nil || node.ID == "" {
				continue
			}

			if podHasPrivilegedContainer(pod) {
				edges = append(edges, CreateEdgeWithProperties(pod, node, "CE_PRIV_MOUNT", map[string]any{
					"Description": "Container in pod is privileged which may allow for mounting the host filesystem.",
					"Reference":   "https://kubehound.io/reference/attacks/CE_PRIV_MOUNT/",
				}))
			}

			if pod.HostPID && podHasPrivilegedContainer(pod) {
				edges = append(edges, CreateEdgeWithProperties(pod, node, "CE_NSENTER", map[string]any{
					"Description": "Container in pod is privileged and has hostPID enabled which may allow for escaping the container and executing commands on the host using nsenter.",
					"Reference":   "https://kubehound.io/reference/attacks/CE_NSENTER/",
				}))
			}

			if isSysPtraceVulnerable(pod) {
				edges = append(edges, CreateEdgeWithProperties(pod, node, "CE_SYS_PTRACE", map[string]any{
					"Description": "Container in pod has CAP_SYS_PTRACE, and CAP_SYS_ADMIN capabilities, and has hostPID: True, or is privileged which allows for tracing and debugging of processes, and can be used to escape the container by attaching to processes running on the host.",
					"Reference":   "https://kubehound.io/reference/attacks/CE_SYS_PTRACE/",
				}))
			}

			if mountPath, ok := podHasCriticalProcfsMount(pod); ok {
				edges = append(edges, CreateEdgeWithProperties(pod, node, "CE_UMH_CORE_PATTERN", map[string]any{
					"Description": "Container in pod has a hostPath volume mount to a critical procfs path which may allow for container escape via usermode helper pattern. Note: this check does not verify if the container is running as the root user, which will likely be required to write to the /proc/sys/kernel/core_pattern file. Mount path: " + mountPath,
					"Reference":   "https://kubehound.io/reference/attacks/CE_UMH_CORE_PATTERN/",
				}))
			}

			if hostPath := podHasSocketHostPath(pod); hostPath != "" {
				edges = append(edges, CreateEdgeWithProperties(pod, node, "MOUNTED_CONTAINER_SOCKET", map[string]any{
					"Description": "Container in pod has a hostPath volume mount to a path that potentially contains a container socket: " + hostPath + ". ",
					"Reference":   "https://kubehound.io/reference/attacks/EXPLOIT_CONTAINERD_SOCK/",
				}))
			}
		}
	}
	return edges
}

func podHasPrivilegedContainer(pod *nodes.PodCore) bool {
	if pod == nil {
		return false
	}
	for _, container := range pod.Containers {
		if MapBool(container, "privileged") {
			return true
		}
	}
	return false
}

func isSysPtraceVulnerable(pod *nodes.PodCore) bool {
	if pod == nil {
		return false
	}
	if podHasPrivilegedContainer(pod) {
		return true
	}
	if pod.HostPID && HasCapability(*pod, "CAP_SYS_PTRACE") && HasCapability(*pod, "CAP_SYS_ADMIN") {
		return true
	}
	return false
}

func podHasSocketHostPath(pod *nodes.PodCore) string {
	if pod == nil {
		return ""
	}
	for _, volume := range pod.Volumes {
		hostPath := MapString(volume, "hostPath")
		if hostPath == "" {
			continue
		}
		if strings.HasSuffix(hostPath, ".sock") {
			return hostPath
		}
	}
	return ""
}

func podHasCriticalProcfsMount(pod *nodes.PodCore) (string, bool) {
	if pod == nil {
		return "", false
	}
	critical := map[string]struct{}{
		"/proc":            {},
		"/proc/sys":        {},
		"/proc/sys/kernel": {},
	}
	for _, volume := range pod.Volumes {
		hostPath := MapString(volume, "hostPath")
		if hostPath == "" {
			continue
		}
		if _, ok := critical[hostPath]; !ok {
			continue
		}
		for _, container := range pod.Containers {
			for _, mountEntry := range MapSlice(container, "volumeMounts") {
				mountMap, ok := mountEntry.(map[string]any)
				if !ok {
					continue
				}
				if MapBool(mountMap, "readOnly") {
					continue
				}
				mountPath := MapString(mountMap, "mountPath")
				if mountPath != "" {
					return mountPath, true
				}
			}
		}
	}
	return "", false
}

func init() {
	RegisterEdgeRule(attackEdgesRule{})
}
