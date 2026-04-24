package security

import (
	"strings"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes"
)

type capabilityInfo struct {
	Description string
	Reference   string
}

var capabilityDescriptions = map[string]capabilityInfo{
	"CAP_SYS_ADMIN": {
		Description: "Container in pod has CAP_SYS_ADMIN capability which is a powerful capability that can allow for a wide range of actions, including privilege escalation and container escape.",
		Reference:   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_admin",
	},
	"CAP_NET_ADMIN": {
		Description: "Container in pod has CAP_NET_ADMIN capability which allows for network administration tasks and can be used for intercepting network traffic or modifying network configurations.",
		Reference:   "",
	},
	"CAP_SYS_MODULE": {
		Description: "Container in pod has CAP_SYS_MODULE capability which allows for loading and unloading kernel modules, and can be used for installing custom kernel modules.",
		Reference:   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_module",
	},
	"CAP_SYS_PTRACE": {
		Description: "Container in pod has CAP_SYS_PTRACE capability which allows for tracing and debugging of processes, and can be used for stealing sensitive information from other processes or performing code injection attacks.",
		Reference:   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_ptrace",
	},
	"CAP_SYS_RAWIO": {
		Description: "Container in pod has CAP_SYS_RAWIO capability which allows for raw I/O operations, and can be used for malicious purposes such as bypassing security controls or accessing sensitive data on the host.",
		Reference:   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_rawio",
	},
}

type capabilityEdgesRule struct{}

func (r capabilityEdgesRule) Name() string { return "capabilities" }

func (r capabilityEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
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
			for _, capAdd := range pod.CapabilitiesAdd {
				norm := framework.NormalizeCapability(capAdd)
				info, ok := capabilityDescriptions[norm]
				if !ok {
					continue
				}
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, norm, map[string]any{
					"Description": info.Description,
					"Reference":   info.Reference,
				}))
			}
		}
	}
	return edges
}

type containerEscapeRule struct{}

func (r containerEscapeRule) Name() string { return "container_escapes" }

func (r containerEscapeRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
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
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "CE_PRIV_MOUNT", map[string]any{
					"Description": "Container in pod is privileged which may allow for mounting the host filesystem.",
					"Reference":   "https://kubehound.io/reference/attacks/CE_PRIV_MOUNT/",
				}))
			}

			if ceNsEnterCheck(pod) {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "CE_NSENTER", map[string]any{
					"Description": "Container in pod is privileged and has hostPID enabled which may allow for escaping the container and executing commands on the host using nsenter.",
					"Reference":   "https://kubehound.io/reference/attacks/CE_NSENTER/",
				}))
			}

			if ceSysPtraceCheck(pod) {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "CE_SYS_PTRACE", map[string]any{
					"Description": "Container in pod has CAP_SYS_PTRACE, and CAP_SYS_ADMIN capabilities, and has hostPID: True, or is privileged which allows for tracing and debugging of processes, and can be used to escape the container by attaching to processes running on the host.",
					"Reference":   "https://kubehound.io/reference/attacks/CE_SYS_PTRACE/",
				}))
			}

			if mountPath, ok := ceUmhCorePatternCheck(pod); ok {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "CE_UMH_CORE_PATTERN", map[string]any{
					"Description": "Container in pod has a hostPath volume mount to a critical procfs path which may allow for container escape via usermode helper pattern. Note: this check does not verify if the container is running as the root user, which will likely be required to write to the /proc/sys/kernel/core_pattern file. Mount path: " + mountPath,
					"Reference":   "https://kubehound.io/reference/attacks/CE_UMH_CORE_PATTERN/",
				}))
			}

			if hostPath := podHasSocketHostPath(pod); hostPath != "" {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "MOUNT_CONTAINER_SOCKET", map[string]any{
					"Description": "Container in pod has a hostPath volume mount to a path that potentially contains a container socket: " + hostPath + ". ",
					"Reference":   "https://kubehound.io/reference/attacks/EXPLOIT_CONTAINERD_SOCK/",
				}))
			}

			if hostPath, ok := ceVarLogSymlinkCheck(pod); ok {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "CE_VAR_LOG_SYMLINK", map[string]any{
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
	return pod.HostPID && framework.HasCapability(*pod, "CAP_SYS_PTRACE") && framework.HasCapability(*pod, "CAP_SYS_ADMIN")
}

func podHasSocketHostPath(pod *nodes.Pod) string {
	if pod == nil {
		return ""
	}
	for _, volume := range pod.Volumes {
		hostPath := volume.HostPath
		if hostPath != "" && strings.HasSuffix(hostPath, ".sock") {
			return hostPath
		}
	}
	return ""
}

func ceUmhCorePatternCheck(pod *nodes.Pod) (string, bool) {
	if pod == nil {
		return "", false
	}
	critical := map[string]struct{}{"/proc": {}, "/proc/sys": {}, "/proc/sys/kernel": {}}
	for _, volume := range pod.Volumes {
		hostPath := volume.HostPath
		if _, ok := critical[hostPath]; !ok {
			continue
		}
		for _, container := range pod.Containers {
			for _, mount := range container.VolumeMounts {
				if !mount.ReadOnly && mount.MountPath != "" {
					return mount.MountPath, true
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
		if container.RunAsNonRoot || !container.Privileged {
			return "", false
		}
	}
	for _, volume := range pod.Volumes {
		hostPath := volume.HostPath
		if hostPath == "/var/log" || hostPath == "/var" {
			return hostPath, true
		}
	}
	return "", false
}
