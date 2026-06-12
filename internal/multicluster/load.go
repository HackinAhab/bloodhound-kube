package multicluster

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadConfig reads and parses a multi-cluster YAML config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read clusters config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse clusters config: %w", err)
	}
	return &cfg, nil
}

// ExpandEnvVars expands ${VAR} references in token fields. Returns an error if
// a referenced env var is unset or empty.
func ExpandEnvVars(cfg *Config) error {
	for i := range cfg.Clusters {
		expanded, err := expandToken(cfg.Clusters[i].Name, cfg.Clusters[i].Token)
		if err != nil {
			return err
		}
		cfg.Clusters[i].Token = expanded
	}
	return nil
}

func expandToken(clusterName, token string) (string, error) {
	if token == "" || !strings.Contains(token, "${") {
		return token, nil
	}
	var expandErr error
	expanded := os.Expand(token, func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			expandErr = fmt.Errorf("cluster %q: env var ${%s} is not set or empty", clusterName, key)
		}
		return val
	})
	return expanded, expandErr
}

// Validate checks structural correctness of the raw config before defaults are
// applied.
func Validate(cfg *Config) error {
	if len(cfg.Clusters) == 0 {
		return fmt.Errorf("clusters config must define at least one cluster")
	}
	seen := make(map[string]bool)
	seenOutputFile := make(map[string]bool)
	for _, e := range cfg.Clusters {
		if e.Name == "" {
			return fmt.Errorf("all cluster entries must have a name")
		}
		if seen[e.Name] {
			return fmt.Errorf("duplicate cluster name %q", e.Name)
		}
		seen[e.Name] = true
		if e.Kubeconfig != "" && e.Server != "" {
			return fmt.Errorf("cluster %q: kubeconfig and server are mutually exclusive", e.Name)
		}
		if (e.Server != "") != (e.Token != "") {
			return fmt.Errorf("cluster %q: server and token must be set together", e.Name)
		}
		if e.AllNamespaces != nil && *e.AllNamespaces && e.Namespace != "" {
			return fmt.Errorf("cluster %q: allNamespaces and namespace are mutually exclusive", e.Name)
		}
		if e.OutputFile != "" {
			if seenOutputFile[e.OutputFile] {
				return fmt.Errorf("cluster %q: duplicate outputFile %q — each cluster must write to a unique path", e.Name, e.OutputFile)
			}
			seenOutputFile[e.OutputFile] = true
		}
	}

	// Reject interactive CRD discovery in multi-cluster mode.
	// Require acceptCRDs: true or a discoveryAllowlist so the run is fully
	// automated.
	if len(cfg.Clusters) > 1 {
		for _, e := range cfg.Clusters {
			effectiveAcceptCRDs := (e.AcceptCRDs != nil && *e.AcceptCRDs) || cfg.Defaults.AcceptCRDs
			effectiveAllowlist := e.DiscoveryAllowlist
			if effectiveAllowlist == "" {
				effectiveAllowlist = cfg.Defaults.DiscoveryAllowlist
			}
			if !effectiveAcceptCRDs && effectiveAllowlist == "" {
				effectiveScope := strings.TrimSpace(strings.ToLower(e.Scope))
				if effectiveScope == "" {
					effectiveScope = strings.TrimSpace(strings.ToLower(cfg.Defaults.Scope))
				}
				if effectiveScope == "all" || effectiveScope == "" {
					return fmt.Errorf(
						"cluster %q: interactive CRD discovery is not supported in multi-cluster mode; "+
							"set acceptCRDs: true (or in defaults) to accept all CRDs, or provide a "+
							"discoveryAllowlist to control which resources are collected",
						e.Name,
					)
				}
			}
		}
	}

	return nil
}

// ApplyDefaults returns a fully-resolved slice of ClusterEntry values with
// each cluster's missing fields filled in from cfg.Defaults. Callers should
// use this slice rather than cfg.Clusters directly.
func ApplyDefaults(cfg *Config) []ClusterEntry {
	d := cfg.Defaults
	out := make([]ClusterEntry, len(cfg.Clusters))
	for i, e := range cfg.Clusters {
		if e.ClusterType == "" {
			e.ClusterType = d.ClusterType
		}
		if e.Scope == "" {
			e.Scope = d.Scope
		}
		if e.Namespace == "" {
			e.Namespace = d.Namespace
		}
		if e.DiscoveryAllowlist == "" {
			e.DiscoveryAllowlist = d.DiscoveryAllowlist
		}
		if e.OutputDir == "" {
			e.OutputDir = d.OutputDir
		}
		if e.AllNamespaces == nil {
			e.AllNamespaces = boolPtr(d.AllNamespaces)
		}
		if e.Redacted == nil {
			e.Redacted = boolPtr(d.Redacted)
		}
		if e.AcceptCRDs == nil {
			e.AcceptCRDs = boolPtr(d.AcceptCRDs)
		}
		if e.Concurrency == 0 {
			e.Concurrency = d.Concurrency
		}
		if e.PaginateLimit == 0 {
			e.PaginateLimit = d.PaginateLimit
		}
		out[i] = e
	}
	return out
}

func boolPtr(b bool) *bool { return &b }
