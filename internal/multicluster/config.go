package multicluster

// Config is the top-level structure for a multi-cluster YAML config file.
type Config struct {
	Defaults ClusterDefaults `yaml:"defaults"`
	Clusters []ClusterEntry  `yaml:"clusters"`
}

// ClusterDefaults holds fallback values applied to any cluster entry that
// does not specify that field explicitly.
type ClusterDefaults struct {
	Scope              string `yaml:"scope"`
	AllNamespaces      bool   `yaml:"allNamespaces"`
	Namespace          string `yaml:"namespace"`
	Redacted           bool   `yaml:"redacted"`
	AcceptCRDs         bool   `yaml:"acceptCRDs"`
	Concurrency        int    `yaml:"concurrency"`
	PaginateLimit      int    `yaml:"paginateLimit"`
	DiscoveryAllowlist string `yaml:"discoveryAllowlist"`
	ClusterType        string `yaml:"clusterType"`
	OutputDir          string `yaml:"outputDir"`
	ClusterConcurrency int    `yaml:"clusterConcurrency"`
}

// ClusterEntry describes a single cluster target. Boolean fields use pointers
// so that an explicit false can be distinguished from "not set, inherit default".
type ClusterEntry struct {
	Name               string `yaml:"name"`
	Kubeconfig         string `yaml:"kubeconfig"`
	Server             string `yaml:"server"`
	Token              string `yaml:"token"`
	ClusterType        string `yaml:"clusterType"`
	Scope              string `yaml:"scope"`
	AllNamespaces      *bool  `yaml:"allNamespaces"`
	Namespace          string `yaml:"namespace"`
	Redacted           *bool  `yaml:"redacted"`
	AcceptCRDs         *bool  `yaml:"acceptCRDs"`
	Concurrency        int    `yaml:"concurrency"`
	PaginateLimit      int    `yaml:"paginateLimit"`
	DiscoveryAllowlist string `yaml:"discoveryAllowlist"`
	OutputFile         string `yaml:"outputFile"`
	OutputDir          string `yaml:"outputDir"`
}
