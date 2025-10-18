package collector

// This file contains type definitions for Kubernetes workload resources:
// Deployments, StatefulSets, DaemonSets, Pods, and related structures.

type Deployment struct {
	CommonResourceMeta
	Spec DeploymentSpec `json:"spec"`
}

type DeploymentSpec struct {
	Selector        map[string]string `json:"selector,omitempty"`
	ContainerImages []string          `json:"container_images,omitempty"`
	SecurityContext *SecurityContext  `json:"security_context,omitempty"`
}

type StatefulSet struct {
	CommonResourceMeta
	ServiceName              string            `json:"service_name"`
	ContainerImages          []string          `json:"container_images,omitempty"`
	Selector                 map[string]string `json:"selector,omitempty"`
	VolumeClaimTemplateNames []string          `json:"volume_claim_template_names,omitempty"`
	SecurityContext          *SecurityContext  `json:"security_context,omitempty"`
}

type DaemonSet struct {
	CommonResourceMeta
	DesiredNumber   int32             `json:"desired_number"`
	CurrentNumber   int32             `json:"current_number"`
	ReadyNumber     int32             `json:"ready_number"`
	UpdatedNumber   int32             `json:"updated_number"`
	AvailableNumber int32             `json:"available_number"`
	ContainerImages []string          `json:"container_images,omitempty"`
	Selector        map[string]string `json:"selector,omitempty"`
}

type Pod struct {
	CommonResourceMeta
	NodeName        string           `json:"node_name,omitempty"`
	HostNetwork     bool             `json:"host_network"`
	ContainerImages []string         `json:"container_images,omitempty"`
	SecurityContext *SecurityContext `json:"security_context,omitempty"`
	Containers      []Container      `json:"containers,omitempty"`
	ServiceAccount  string           `json:"service_account,omitempty"`
	ResourceLimits  *ResourceLimits  `json:"resource_limits,omitempty"`
}

type Container struct {
	Name            string           `json:"name"`
	Image           string           `json:"image"`
	SecurityContext *SecurityContext `json:"security_context,omitempty"`
	ResourceLimits
}

type ResourceLimits struct {
	CpuReq   string `json:"cpu_request,omitempty"`
	CpuLimit string `json:"cpu_limit,omitempty"`
	MemReq   string `json:"memory_request,omitempty"`
	MemLimit string `json:"memory_limit,omitempty"`
}

// Security-related types used by workloads
type SecurityContext struct {
	RunAsUser         *int64             `json:"run_as_user,omitempty"`
	RunAsGroup        *int64             `json:"run_as_group,omitempty"`
	RunAsNonRoot      *bool              `json:"run_as_non_root,omitempty"`
	FSGroup           *int64             `json:"fs_group,omitempty"`
	AllowPrivEsc      *bool              `json:"allow_priv_esc,omitempty"`
	SeccompProfile    *SeccompProfile    `json:"seccomp_profile,omitempty"`
	LinuxCapabilities *LinuxCapabilities `json:"linux_capabilities,omitempty"`
}

type SeccompProfile struct {
	Type             string `json:"seccomp_type,omitempty"`
	LocalhostProfile string `json:"localhost_profile,omitempty"`
}

type LinuxCapabilities struct {
	Add  []string `json:"capabilities_add,omitempty"`
	Drop []string `json:"capabilities_drop,omitempty"`
}
