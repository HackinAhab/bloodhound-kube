package edges

import (
	"strings"

	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

type containerEscapeRule struct{}

func (r containerEscapeRule) Name() string {
	return "container_escapes"
}

func (r containerEscapeRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
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

			if ceNsEnterCheck(pod) {
				edges = append(edges, CreateEdgeWithProperties(pod, node, "CE_NSENTER", map[string]any{
					"Description": "Container in pod is privileged and has hostPID enabled which may allow for escaping the container and executing commands on the host using nsenter.",
					"Reference":   "https://kubehound.io/reference/attacks/CE_NSENTER/",
				}))
			}

			if ceSysPtraceCheck(pod) {
				edges = append(edges, CreateEdgeWithProperties(pod, node, "CE_SYS_PTRACE", map[string]any{
					"Description": "Container in pod has CAP_SYS_PTRACE, and CAP_SYS_ADMIN capabilities, and has hostPID: True, or is privileged which allows for tracing and debugging of processes, and can be used to escape the container by attaching to processes running on the host.",
					"Reference":   "https://kubehound.io/reference/attacks/CE_SYS_PTRACE/",
				}))
			}

			if mountPath, ok := ceUmhCorePatternCheck(pod); ok {
				edges = append(edges, CreateEdgeWithProperties(pod, node, "CE_UMH_CORE_PATTERN", map[string]any{
					"Description": "Container in pod has a hostPath volume mount to a critical procfs path which may allow for container escape via usermode helper pattern. Note: this check does not verify if the container is running as the root user, which will likely be required to write to the /proc/sys/kernel/core_pattern file. Mount path: " + mountPath,
					"Reference":   "https://kubehound.io/reference/attacks/CE_UMH_CORE_PATTERN/",
				}))
			}

			if hostPath := podHasSocketHostPath(pod); hostPath != "" {
				edges = append(edges, CreateEdgeWithProperties(pod, node, "MOUNT_CONTAINER_SOCKET", map[string]any{
					"Description": "Container in pod has a hostPath volume mount to a path that potentially contains a container socket: " + hostPath + ". ",
					"Reference":   "https://kubehound.io/reference/attacks/EXPLOIT_CONTAINERD_SOCK/",
				}))
			}

			if hostPath, ok := ceVarLogSymlinkCheck(pod); ok {
				edges = append(edges, CreateEdgeWithProperties(pod, node, "CE_VAR_LOG_SYMLINK", map[string]any{
					"Description": "Container in pod has a hostPath volume mount to /var/log or /var which may allow for container escape via log file symlink attack. Note: this check does not verify if the container is running as the root user, which will likely be required to create symlinks to sensitive host files. Host path: " + hostPath,
					"Reference":   "https://kubehound.io/reference/attacks/CE_VAR_LOG_SYMLINK/",
				}))
			}
		}
	}
	return edges
}

func podHasPrivilegedContainer(pod *nodes.Pod) bool {
	if pod == nil {
		return false
	}
	for _, container := range pod.Containers {
		if container.Privileged {
			return true
		}
	}
	return false
}

func ceSysPtraceCheck(pod *nodes.Pod) bool {
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

func podHasSocketHostPath(pod *nodes.Pod) string {
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

func ceUmhCorePatternCheck(pod *nodes.Pod) (string, bool) {
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
			for _, mount := range container.VolumeMounts {
				if mount.ReadOnly {
					continue
				}
				mountPath := mount.MountPath
				if mountPath != "" {
					return mountPath, true
				}
			}
		}
	}
	return "", false
}

func ceNsEnterCheck(pod *nodes.Pod) bool {
	if pod == nil {
		return false
	}
	return pod.HostPID && podHasPrivilegedContainer(pod)
}

func ceVarLogSymlinkCheck(pod *nodes.Pod) (string, bool) {
	if pod == nil {
		return "", false
	}
	for _, container := range pod.Containers {
		// This check is not perfect, it doesn't verify if the container is actually running as root.
		if container.RunAsNonRoot || !container.Privileged {
			return "", false
		}
	}
	for _, volume := range pod.Volumes {
		hostPath := MapString(volume, "hostPath")
		if hostPath == "/var/log" || hostPath == "/var" {
			return hostPath, true
		}
	}
	return "", false
}

func init() {
	RegisterEdgeRule(containerEscapeRule{})
}
