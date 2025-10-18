package collector

// This file contains type definitions for Kubernetes networking resources:
// Services, Ingresses, NetworkPolicies, Gateways, and related structures.

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
