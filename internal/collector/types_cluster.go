package collector

// This file contains type definitions for cluster-scoped Kubernetes resources:
// Nodes, CRDs, and related structures.

type Node struct {
	CommonResourceMeta
	Hostname         string `json:"hostname"`
	InternalIP       string `json:"internal_ip"`
	ExternalIP       string `json:"external_ip,omitempty"`
	PodCIDR          string `json:"pod_cidr,omitempty"`
	KubeletVersion   string `json:"kubelet_version"`
	ContainerRuntime string `json:"container_runtime"`
	OSImage          string `json:"os_image"`
	KernelVersion    string `json:"kernel_version"`
	Architecture     string `json:"architecture"`
	OperatingSystem  string `json:"operating_system"`
	Unschedulable    bool   `json:"unschedulable"`
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
