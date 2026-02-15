package report

import (
	"time"

	"bloodhound-kube/internal/utils"
)

// Config holds the configuration for report generation
type Config struct {
	InputFile         string
	OutputPrefix      string
	ReportTypes       []string
	Format            string
	TrustedRegistries string
	Verbose           bool
}

// Generator handles report generation
type Generator struct {
	config Config
	log    utils.Logger
	data   *CollectedData
}

// Report represents a generated report
type Report struct {
	Type  string
	Count int
	Data  any
}

// CollectedData holds the parsed JSONL data organized by type
type CollectedData struct {
	Namespaces map[string]*Namespace
}

// Namespace represents a Kubernetes namespace with its resources
type Namespace struct {
	Name            string            `json:"name"`
	Pods            []*Pod            `json:"pods,omitempty"`
	Secrets         []*Secret         `json:"secrets,omitempty"`
	ServiceAccounts []*ServiceAccount `json:"serviceaccounts,omitempty"`
	RBAC            []*RBACResource   `json:"rbac,omitempty"`
}

// Pod represents a Kubernetes pod
type Pod struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	NodeName        string            `json:"node_name,omitempty"`
	HostNetwork     bool              `json:"host_network"`
	ServiceAccount  string            `json:"service_account,omitempty"`
	SecurityContext *SecurityContext  `json:"security_context,omitempty"`
	Containers      []*Container      `json:"containers"`
	CreatedAt       time.Time         `json:"created_at"`
	Labels          map[string]string `json:"labels,omitempty"`
	CPURequest      string            `json:"cpu_request,omitempty"`
	CPULimit        string            `json:"cpu_limit,omitempty"`
	MemoryRequest   string            `json:"memory_request,omitempty"`
	MemoryLimit     string            `json:"memory_limit,omitempty"`
}

// Container represents a container within a pod
type Container struct {
	Name            string           `json:"name"`
	Image           string           `json:"image"`
	SecurityContext *SecurityContext `json:"security_context,omitempty"`
	CPURequest      string           `json:"cpu_request,omitempty"`
	CPULimit        string           `json:"cpu_limit,omitempty"`
	MemoryRequest   string           `json:"memory_request,omitempty"`
	MemoryLimit     string           `json:"memory_limit,omitempty"`
}

// SecurityContext represents security context settings
type SecurityContext struct {
	RunAsUser                *int64        `json:"run_as_user,omitempty"`
	RunAsNonRoot             *bool         `json:"run_as_non_root,omitempty"`
	AllowPrivilegeEscalation *bool         `json:"allow_privilege_escalation,omitempty"`
	Privileged               *bool         `json:"privileged,omitempty"`
	Capabilities             *Capabilities `json:"capabilities,omitempty"`
	SeccompProfile           string        `json:"seccomp_profile,omitempty"`
}

// Capabilities represents Linux capabilities
type Capabilities struct {
	Add  []string `json:"add,omitempty"`
	Drop []string `json:"drop,omitempty"`
}

// Secret represents a Kubernetes secret
type Secret struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	DataKeys  []string          `json:"data_keys,omitempty"`
	Data      map[string]string `json:"data,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// ServiceAccount represents a Kubernetes service account
type ServiceAccount struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Token     string    `json:"token,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// RBACResource represents RBAC resources (Roles, ClusterRoles, etc.)
type RBACResource struct {
	Name      string       `json:"name"`
	Namespace string       `json:"namespace,omitempty"` // Empty for ClusterRoles
	Kind      string       `json:"kind"`
	Rules     []PolicyRule `json:"rules,omitempty"`
	Subjects  []Subject    `json:"subjects,omitempty"`
	RoleRef   *RoleRef     `json:"role_ref,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// PolicyRule represents a policy rule in RBAC
type PolicyRule struct {
	APIGroups     []string `json:"api_groups,omitempty"`
	Resources     []string `json:"resources,omitempty"`
	Verbs         []string `json:"verbs,omitempty"`
	ResourceNames []string `json:"resource_names,omitempty"`
}

// Subject represents a subject in RBAC
type Subject struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// RoleRef represents a role reference in RBAC
type RoleRef struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	APIGroup string `json:"api_group"`
}

// ReportNamespace represents the output format for namespace-based reports
type ReportNamespace struct {
	Namespace string `json:"namespace"`
	Pods      []any  `json:"pods"`
}

// PodReport represents a pod in the report output
type PodReport struct {
	Name       string `json:"name"`
	Containers []any  `json:"containers,omitempty"`
}
