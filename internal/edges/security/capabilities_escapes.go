package security

import (
	"strings"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes/workload"
)

type capabilityInfo struct {
	Description string
	Reference   string
}

var capabilityDescriptions = map[string]capabilityInfo{
	"BHK_CAP_SYS_ADMIN": {
		Description: "Container in pod has CAP_SYS_ADMIN capability which is a powerful capability that can allow for a wide range of actions, including privilege escalation and container escape.",
		Reference:   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_admin",
	},
	"BHK_CAP_NET_ADMIN": {
		Description: "Container in pod has CAP_NET_ADMIN capability which allows for network administration tasks and can be used for intercepting network traffic or modifying network configurations.",
		Reference:   "",
	},
	"BHK_CAP_SYS_MODULE": {
		Description: "Container in pod has CAP_SYS_MODULE capability which allows for loading and unloading kernel modules, and can be used for installing custom kernel modules.",
		Reference:   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_module",
	},
	"BHK_CAP_SYS_PTRACE": {
		Description: "Container in pod has CAP_SYS_PTRACE capability which allows for tracing and debugging of processes, and can be used for stealing sensitive information from other processes or performing code injection attacks.",
		Reference:   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_ptrace",
	},
	"BHK_CAP_SYS_RAWIO": {
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
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "BHK_CE_PRIV_MOUNT", map[string]any{
					"Description": "Container in pod is privileged which may allow for mounting the host filesystem.",
					"Reference":   "https://kubehound.io/reference/attacks/CE_PRIV_MOUNT/",
				}))
			}

			if ceNsEnterCheck(pod) {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "BHK_CE_NSENTER", map[string]any{
					"Description": "Container in pod is privileged and has hostPID enabled which may allow for escaping the container and executing commands on the host using nsenter.",
					"Reference":   "https://kubehound.io/reference/attacks/CE_NSENTER/",
				}))
			}

			if ceSysPtraceCheck(pod) {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "BHK_CE_SYS_PTRACE", map[string]any{
					"Description": "Container in pod has CAP_SYS_PTRACE, and CAP_SYS_ADMIN capabilities, and has hostPID: True, or is privileged which allows for tracing and debugging of processes, and can be used to escape the container by attaching to processes running on the host.",
					"Reference":   "https://kubehound.io/reference/attacks/CE_SYS_PTRACE/",
				}))
			}

			if mountPath, ok := ceUmhCorePatternCheck(pod); ok {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "BHK_CE_UMH_CORE_PATTERN", map[string]any{
					"Description": "Container in pod has a hostPath volume mount to a critical procfs path which may allow for container escape via usermode helper pattern. Note: this check does not verify if the container is running as the root user, which will likely be required to write to the /proc/sys/kernel/core_pattern file. Mount path: " + mountPath,
					"Reference":   "https://kubehound.io/reference/attacks/CE_UMH_CORE_PATTERN/",
				}))
			}

			if hostPath := podHasSocketHostPath(pod); hostPath != "" {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "BHK_MOUNT_CONTAINER_SOCKET", map[string]any{
					"Description": "Container in pod has a hostPath volume mount to a path that potentially contains a container socket: " + hostPath + ". ",
					"Reference":   "https://kubehound.io/reference/attacks/EXPLOIT_CONTAINERD_SOCK/",
				}))
			}

			if hostPath, ok := ceVarLogSymlinkCheck(pod); ok {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "BHK_CE_VAR_LOG_SYMLINK", map[string]any{
					"Description": "Container in pod has a hostPath volume mount to /var/log or /var which may allow for container escape via log file symlink attack. Note: this check does not verify if the container is running as the root user, which will likely be required to create symlinks to sensitive host files. Host path: " + hostPath,
					"Reference":   "https://kubehound.io/reference/attacks/CE_VAR_LOG_SYMLINK/",
				}))
			}

			if ceHostIPCCheck(pod) {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "BHK_CE_HOST_IPC", map[string]any{
					"Description": "Pod has hostIPC: true with a privileged container or CAP_SYS_ADMIN. The container shares the host IPC namespace, allowing access to host shared memory, semaphores, and message queues. Combined with privilege, this enables IPC-based process injection or triggering usermode helper patterns.",
					"Reference":   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/docker-security/docker-breakout-privilege-escalation/",
				}))
			}

			if ceHostNetworkCheck(pod) {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "BHK_CE_HOST_NETWORK", map[string]any{
					"Description": "Pod has hostNetwork: true, sharing the host network namespace. The container can bind to any host port, intercept host-level traffic, and reach network services accessible only from the node.",
					"Reference":   "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/docker-security/docker-breakout-privilege-escalation/",
				}))
			}

			if ceShareProcNsCheck(pod) {
				edges = append(edges, framework.CreateEdgeWithProperties(pod, node, "BHK_CE_SHARE_PROC_NS", map[string]any{
					"Description": "Pod has shareProcessNamespace: true with a privileged container or CAP_SYS_PTRACE. All containers in the pod share a single PID namespace; a privileged container can ptrace other containers' processes to inject code or steal secrets from memory.",
					"Reference":   "https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace/",
				}))
			}
		}
	}
	return edges
}

func podHasPrivilegedContainer(pod *workload.Pod) bool {
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

func ceSysPtraceCheck(pod *workload.Pod) bool {
	if pod == nil {
		return false
	}
	if podHasPrivilegedContainer(pod) {
		return true
	}
	return pod.HostPID && framework.HasCapability(*pod, "CAP_SYS_PTRACE") && framework.HasCapability(*pod, "CAP_SYS_ADMIN")
}

func podHasSocketHostPath(pod *workload.Pod) string {
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

var ceUmhCriticalPaths = map[string]struct{}{"/proc": {}, "/proc/sys": {}, "/proc/sys/kernel": {}}

func ceUmhCorePatternCheck(pod *workload.Pod) (string, bool) {
	if pod == nil {
		return "", false
	}
	for _, volume := range pod.Volumes {
		hostPath := volume.HostPath
		if _, ok := ceUmhCriticalPaths[hostPath]; !ok {
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

func ceNsEnterCheck(pod *workload.Pod) bool {
	if pod == nil {
		return false
	}
	return pod.HostPID && podHasPrivilegedContainer(pod)
}

func ceHostIPCCheck(pod *workload.Pod) bool {
	if pod == nil || !pod.HostIPC {
		return false
	}
	return podHasPrivilegedContainer(pod) || framework.HasCapability(*pod, "CAP_SYS_ADMIN")
}

func ceHostNetworkCheck(pod *workload.Pod) bool {
	if pod == nil || !pod.HostNetwork {
		return false
	}
	return podHasPrivilegedContainer(pod) ||
		framework.HasCapability(*pod, "CAP_NET_ADMIN") ||
		framework.HasCapability(*pod, "CAP_NET_RAW")
}

func ceShareProcNsCheck(pod *workload.Pod) bool {
	if pod == nil || pod.ShareProcNs == nil || !*pod.ShareProcNs {
		return false
	}
	return podHasPrivilegedContainer(pod) || framework.HasCapability(*pod, "CAP_SYS_PTRACE")
}

func ceVarLogSymlinkCheck(pod *workload.Pod) (string, bool) {
	if pod == nil {
		return "", false
	}
	hasQualifyingContainer := false
	for _, container := range pod.Containers {
		isRoot := container.Privileged || (container.RunAsUser != nil && *container.RunAsUser == 0)
		if isRoot && !container.RunAsNonRoot {
			hasQualifyingContainer = true
			break
		}
	}
	if !hasQualifyingContainer {
		return "", false
	}
	for _, volume := range pod.Volumes {
		hostPath := volume.HostPath
		if hostPath == "/var/log" || hostPath == "/var" {
			return hostPath, true
		}
	}
	return "", false
}
