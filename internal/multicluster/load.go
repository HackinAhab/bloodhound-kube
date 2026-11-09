package multicluster

import (
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"bloodhound-kube/internal/collector"
	"bloodhound-kube/internal/utils"

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
		expanded, err := ExpandToken(cfg.Clusters[i].Name, cfg.Clusters[i].Token)
		if err != nil {
			return err
		}
		cfg.Clusters[i].Token = expanded
	}
	return nil
}

// ExpandToken expands environment references in one cluster's token.
func ExpandToken(clusterName, token string) (string, error) {
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
		if err := validateEntryTLS(e, cfg.Defaults); err != nil {
			return err
		}
		if _, err := collector.ParseAllowlistEntries(e.DiscoveryAllowlistEntries); err != nil {
			return fmt.Errorf("cluster %q: discoveryAllowlistEntries: %w", e.Name, err)
		}
	}
	if _, err := collector.ParseAllowlistEntries(cfg.Defaults.DiscoveryAllowlistEntries); err != nil {
		return fmt.Errorf("defaults: discoveryAllowlistEntries: %w", err)
	}
	if d := cfg.Defaults; d.CACert != "" && d.InsecureSkipTLSVerify != nil && *d.InsecureSkipTLSVerify {
		return fmt.Errorf("defaults: caCert and insecureSkipTLSVerify: true are mutually exclusive")
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
			hasInline := len(e.DiscoveryAllowlistEntries) > 0 || len(cfg.Defaults.DiscoveryAllowlistEntries) > 0
			if !effectiveAcceptCRDs && effectiveAllowlist == "" && !hasInline {
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
		if e.Context == "" {
			e.Context = d.Context
		}
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
		if len(e.DiscoveryAllowlistEntries) == 0 {
			e.DiscoveryAllowlistEntries = d.DiscoveryAllowlistEntries
		}
		if e.ParseUndefinedNodes == nil {
			e.ParseUndefinedNodes = boolPtr(d.ParseUndefinedNodes)
		}
		// TLS settings are only meaningful for server+token clusters. Inherit
		// each default only when the entry has not chosen the opposing option.
		if e.Server != "" {
			if e.CACert == "" && (e.InsecureSkipTLSVerify == nil || !*e.InsecureSkipTLSVerify) {
				e.CACert = d.CACert
			}
			if e.InsecureSkipTLSVerify == nil && e.CACert == "" {
				e.InsecureSkipTLSVerify = d.InsecureSkipTLSVerify
			}
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

// validateEntryTLS checks caCert/insecureSkipTLSVerify for one cluster entry.
// The effective CA (entry, else default) must be a readable PEM bundle.
func validateEntryTLS(e ClusterEntry, d ClusterDefaults) error {
	insecure := e.InsecureSkipTLSVerify != nil && *e.InsecureSkipTLSVerify
	if e.CACert != "" && insecure {
		return fmt.Errorf("cluster %q: caCert and insecureSkipTLSVerify: true are mutually exclusive", e.Name)
	}
	if e.Server == "" {
		if e.CACert != "" || e.InsecureSkipTLSVerify != nil {
			return fmt.Errorf("cluster %q: caCert and insecureSkipTLSVerify require server and token; kubeconfig clusters take TLS settings from the kubeconfig", e.Name)
		}
		return nil
	}
	ca := e.CACert
	if ca == "" && !insecure {
		ca = d.CACert
	}
	if ca == "" {
		return nil
	}
	pem, err := os.ReadFile(utils.ExpandTildeInPath(ca))
	if err != nil {
		return fmt.Errorf("cluster %q: caCert: %w", e.Name, err)
	}
	if !x509.NewCertPool().AppendCertsFromPEM(pem) {
		return fmt.Errorf("cluster %q: caCert %q contains no valid PEM certificates", e.Name, ca)
	}
	return nil
}
