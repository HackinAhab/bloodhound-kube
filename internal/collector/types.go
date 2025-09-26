package collector

import (
	"time"
)

// This file contains all the data structures used by collectors
// to represent the collected Kubernetes resources.

// CommonResourceMeta contains fields that appear across multiple resource types
type CommonResourceMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"` // Some resources like Nodes don't have namespace
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

type ConfigMap struct {
	CommonResourceMeta
	DataKeys       []string          `json:"data_keys"`
	Data           map[string]string `json:"data,omitempty"`
	BinaryDataKeys []string          `json:"binary_data_keys,omitempty"`
	BinaryData     map[string][]byte `json:"binary_data,omitempty"`
}

type Secret struct {
	CommonResourceMeta
	Type     string            `json:"type"`
	DataKeys []string          `json:"data_keys"`
	Data     map[string]string `json:"data,omitempty"` // Made optional since data might be redacted
}

type Deployment struct {
	CommonResourceMeta
	Spec   DeploymentSpec   `json:"spec"`
	Status DeploymentStatus `json:"status"`
}

type DeploymentSpec struct {
	Replicas                *int32            `json:"replicas,omitempty"`
	Selector                map[string]string `json:"selector,omitempty"`
	StrategyType            string            `json:"strategy_type,omitempty"`
	RevisionHistoryLimit    *int32            `json:"revision_history_limit,omitempty"`
	ProgressDeadlineSeconds *int32            `json:"progress_deadline_seconds,omitempty"`
	ContainerImages         []string          `json:"container_images,omitempty"`
	SecurityContext         *SecurityContext  `json:"security_context,omitempty"`
}

type DeploymentStatus struct {
	Replicas            int32 `json:"replicas"`
	ReadyReplicas       int32 `json:"ready_replicas"`
	AvailableReplicas   int32 `json:"available_replicas"`
	UnavailableReplicas int32 `json:"unavailable_replicas"`
	UpdatedReplicas     int32 `json:"updated_replicas"`
	ObservedGeneration  int64 `json:"observed_generation"`
}

type SecurityContext struct {
	RunAsUser    *int64 `json:"run_as_user,omitempty"`
	RunAsGroup   *int64 `json:"run_as_group,omitempty"`
	RunAsNonRoot *bool  `json:"run_as_non_root,omitempty"`
	FSGroup      *int64 `json:"fs_group,omitempty"`
	AllowPrivEsc *bool  `json:"allow_priv_esc,omitempty"`
}

type CRD struct {
	CommonResourceMeta
	Group      string       `json:"group"`
	Kind       string       `json:"kind"`
	Version    string       `json:"version"`
	Scope      string       `json:"scope"`
	Plural     string       `json:"plural"`
	Singular   string       `json:"singular"`
	ShortNames []string     `json:"short_names,omitempty"`
	Categories []string     `json:"categories,omitempty"`
	Versions   []CRDVersion `json:"versions,omitempty"`
}

type CRDVersion struct {
	Name    string `json:"name"`
	Served  bool   `json:"served"`
	Storage bool   `json:"storage"`
}

type Node struct {
	Name             string            `json:"name"`
	Labels           map[string]string `json:"labels,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	Hostname         string            `json:"hostname"`
	InternalIP       string            `json:"internal_ip"`
	ExternalIP       string            `json:"external_ip,omitempty"`
	PodCIDR          string            `json:"pod_cidr,omitempty"`
	KubeletVersion   string            `json:"kubelet_version"`
	ContainerRuntime string            `json:"container_runtime"`
	OSImage          string            `json:"os_image"`
	KernelVersion    string            `json:"kernel_version"`
	Architecture     string            `json:"architecture"`
	OperatingSystem  string            `json:"operating_system"`
	Unschedulable    bool              `json:"unschedulable"`
	Capacity         NodeResources     `json:"capacity"`
	Allocatable      NodeResources     `json:"allocatable"`
	Taints           []NodeTaint       `json:"taints,omitempty"`
}

type NodeResources struct {
	CPU              string `json:"cpu"`
	Memory           string `json:"memory"`
	EphemeralStorage string `json:"ephemeral_storage"`
	Pods             string `json:"pods"`
}

type NodeTaint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

type Service struct {
	CommonResourceMeta
	Type            string            `json:"type"`
	ClusterIP       string            `json:"cluster_ip,omitempty"`
	ExternalIPs     []string          `json:"external_ips,omitempty"`
	LoadBalancerIP  string            `json:"load_balancer_ip,omitempty"`
	Ports           []ServicePort     `json:"ports,omitempty"`
	Selector        map[string]string `json:"selector,omitempty"`
	SessionAffinity string            `json:"session_affinity,omitempty"`
	ExternalName    string            `json:"external_name,omitempty"` // Only used for ExternalName type
}

type ServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port"`
	NodePort   int32  `json:"node_port,omitempty"`
}

type Ingress struct {
	CommonResourceMeta
	Hosts []string      `json:"hosts,omitempty"`
	Paths []IngressPath `json:"paths,omitempty"`
	TLS   []IngressTLS  `json:"tls,omitempty"`
}

type IngressPath struct {
	Host    string `json:"host"`
	Path    string `json:"path"`
	Service string `json:"service"`
	Port    string `json:"port"`
}

type IngressTLS struct {
	Hosts      []string `json:"hosts"`
	SecretName string   `json:"secret_name"`
}

type Gateway struct {
	CommonResourceMeta
	Listeners []GatewayListener `json:"listeners,omitempty"`
}

type GatewayListener struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Port     int32  `json:"port"`
	Hostname string `json:"hostname,omitempty"`
}

type NetworkPolicy struct {
	CommonResourceMeta
	PodSelector map[string]string          `json:"pod_selector,omitempty"`
	PolicyTypes []string                   `json:"policy_types,omitempty"`
	Ingress     []NetworkPolicyIngressRule `json:"ingress,omitempty"`
	Egress      []NetworkPolicyEgressRule  `json:"egress,omitempty"`
}

type NetworkPolicyIngressRule struct {
	Ports []NetworkPolicyPort `json:"ports,omitempty"`
	From  []NetworkPolicyPeer `json:"from,omitempty"`
}

type NetworkPolicyEgressRule struct {
	Ports []NetworkPolicyPort `json:"ports,omitempty"`
	To    []NetworkPolicyPeer `json:"to,omitempty"`
}

type NetworkPolicyPort struct {
	Protocol string `json:"protocol,omitempty"`
	Port     string `json:"port,omitempty"`
}

type NetworkPolicyPeer struct {
	PodSelector       map[string]string `json:"pod_selector,omitempty"`
	NamespaceSelector map[string]string `json:"namespace_selector,omitempty"`
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

// RBAC Types
type RBACResource struct {
	CommonResourceMeta
	Kind     string        `json:"kind"`
	Rules    []PolicyRule  `json:"rules,omitempty"`
	Subjects []RBACSubject `json:"subjects,omitempty"`
	RoleRef  *RoleRef      `json:"role_ref,omitempty"`
}

type PolicyRule struct {
	APIGroups     []string `json:"api_groups,omitempty"`
	Resources     []string `json:"resources,omitempty"`
	Verbs         []string `json:"verbs,omitempty"`
	ResourceNames []string `json:"resource_names,omitempty"`
}

type RBACSubject struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type RoleRef struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	APIGroup string `json:"api_group"`
}

type StatefulSet struct {
	CommonResourceMeta
	Replicas                 int32             `json:"replicas"`
	ReadyReplicas            int32             `json:"ready_replicas"`
	CurrentReplicas          int32             `json:"current_replicas"`
	UpdatedReplicas          int32             `json:"updated_replicas"`
	ObservedGeneration       int64             `json:"observed_generation"`
	ServiceName              string            `json:"service_name"`
	PodManagementPolicy      string            `json:"pod_management_policy"`
	UpdateStrategyType       string            `json:"update_strategy_type"`
	Partition                *int32            `json:"partition,omitempty"`
	ContainerImages          []string          `json:"container_images,omitempty"`
	Selector                 map[string]string `json:"selector,omitempty"`
	VolumeClaimTemplateNames []string          `json:"volume_claim_template_names,omitempty"`
	SecurityContext          *SecurityContext  `json:"security_context,omitempty"`
}

// OpenShift Types
type Route struct {
	CommonResourceMeta
	Host    string    `json:"host"`
	Path    string    `json:"path,omitempty"`
	Service string    `json:"service"`
	Port    string    `json:"port,omitempty"`
	TLS     *RouteTLS `json:"tls,omitempty"`
}

type RouteTLS struct {
	Termination string `json:"termination"`
	Certificate string `json:"certificate,omitempty"`
	Key         string `json:"key,omitempty"`
}

type Project struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	DisplayName string            `json:"display_name,omitempty"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status"`
}

type Image struct {
	Name                 string            `json:"name"`
	Labels               map[string]string `json:"labels,omitempty"`
	Annotations          map[string]string `json:"annotations,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	DockerImageReference string            `json:"docker_image_reference"`
	DockerImageManifest  string            `json:"docker_image_manifest,omitempty"`
	DockerImageMetadata  string            `json:"docker_image_metadata,omitempty"`
}
