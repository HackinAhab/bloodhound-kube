package edges

import (
	"fmt"
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
	for key, value := range props {
		properties[key] = value
	}
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

func CreateEdgeVia(start, end, via nodes.EdgeNode, kind string) model.BloodHoundEdge {
	props := map[string]any{
		"via_id":   via.EdgeID(),
		"via_kind": via.EdgeKind(),
		"via_name": via.EdgeName(),
	}
	return CreateEdgeWithProperties(start, end, kind, props)
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

func HasCapability(pod nodes.PodCore, capability string) bool {
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

func LabelsMatchSelector(labels map[string]any, selector map[string]any) bool {
	if len(selector) == 0 {
		return true
	}
	for key, value := range selector {
		if labels == nil {
			return false
		}
		if labels[key] != value {
			return false
		}
	}
	return true
}

func IsSubset(subset map[string]any, superset map[string]any) bool {
	return LabelsMatchSelector(superset, subset)
}

func MapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if value, ok := m[key]; ok {
		if s, ok := value.(string); ok {
			return s
		}
	}
	return ""
}

func MapBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	value, ok := m[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

func MapSlice(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	if value, ok := m[key]; ok {
		if s, ok := value.([]any); ok {
			return s
		}
	}
	return nil
}

func MapMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if value, ok := m[key]; ok {
		if v, ok := value.(map[string]any); ok {
			return v
		}
	}
	return nil
}

func MapNumber(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	value, ok := m[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}

func joinNamespaceKey(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return fmt.Sprintf("%s/%s", namespace, name)
}
