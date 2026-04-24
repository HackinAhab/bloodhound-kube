package framework

import (
	"strings"

	"bloodhound-kube/internal/nodes"
)

func LabelsMatchOnly(labels map[string]any, selector map[string]string) bool {
	if len(selector) == 0 {
		return true
	}
	if labels == nil {
		return false
	}
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}

func NormalizeCapability(capability string) string {
	if strings.HasPrefix(capability, "CAP_") {
		return capability
	}
	if capability == "" {
		return ""
	}
	return "CAP_" + capability
}

func HasCapability(pod nodes.Pod, capability string) bool {
	if capability == "" {
		return false
	}
	for _, capAdd := range pod.CapabilitiesAdd {
		if NormalizeCapability(capAdd) == capability {
			return true
		}
	}
	return false
}

func HostPathMatchesAny(hostPath string, checkPaths []string) bool {
	if hostPath == "" {
		return false
	}
	for _, path := range checkPaths {
		if hostPath == path {
			return true
		}
		if strings.HasPrefix(hostPath, path+"/") {
			return true
		}
	}
	return false
}
