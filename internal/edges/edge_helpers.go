package edges

import (
	"maps"
	"strings"

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

func CreateEdge(start, end nodes.EdgeNode, kind string) model.BloodHoundEdge {
	return CreateEdgeWithProperties(start, end, kind, nil)
}

func CreateEdgeWithProperties(start, end nodes.EdgeNode, kind string, props map[string]any) model.BloodHoundEdge {
	properties := map[string]any{}
	maps.Copy(properties, props)
	if len(properties) == 0 {
		properties = nil
	}
	return model.BloodHoundEdge{
		Start: model.BloodHoundEdgeRef{
			MatchBy: "id",
			Value:   start.EdgeID(),
			Kind:    start.EdgeKind(),
		},
		End: model.BloodHoundEdgeRef{
			MatchBy: "id",
			Value:   end.EdgeID(),
			Kind:    end.EdgeKind(),
		},
		Kind:       kind,
		Properties: properties,
	}
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

func labelsMatchOnly(labels map[string]any, selector map[string]string) bool {
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
