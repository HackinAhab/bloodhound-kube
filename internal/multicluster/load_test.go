package multicluster

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	t.Run("minimal config", func(t *testing.T) {
		path := writeTempYAML(t, `
clusters:
  - name: dev
`)
		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Clusters) != 1 || cfg.Clusters[0].Name != "dev" {
			t.Fatalf("unexpected clusters: %+v", cfg.Clusters)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path/clusters.yaml")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path := writeTempYAML(t, ":\t:invalid")
		_, err := LoadConfig(path)
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "empty clusters",
			cfg:     Config{},
			wantErr: "at least one cluster",
		},
		{
			name:    "missing name",
			cfg:     Config{Clusters: []ClusterEntry{{}}},
			wantErr: "must have a name",
		},
		{
			name: "duplicate name",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "prod"},
				{Name: "prod"},
			}},
			wantErr: "duplicate cluster name",
		},
		{
			name: "kubeconfig and server both set",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", Kubeconfig: "~/.kube/config", Server: "https://k8s"},
			}},
			wantErr: "mutually exclusive",
		},
		{
			name: "server without token",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", Server: "https://k8s"},
			}},
			wantErr: "server and token must be set together",
		},
		{
			name: "token without server",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", Token: "mytoken"},
			}},
			wantErr: "server and token must be set together",
		},
		{
			name: "allNamespaces and namespace conflict",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", AllNamespaces: boolPtr(true), Namespace: "prod"},
			}},
			wantErr: "mutually exclusive",
		},
		{
			name: "valid minimal",
			cfg:  Config{Clusters: []ClusterEntry{{Name: "dev"}}},
		},
		{
			name: "valid server+token",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", Server: "https://k8s", Token: "tok"},
			}},
		},
		// Duplicate outputFile: two clusters sharing an explicit output path
		// would overwrite each other.
		{
			name: "duplicate outputFile",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", OutputFile: "/out/shared.jsonl"},
				{Name: "c2", OutputFile: "/out/shared.jsonl"},
			}},
			wantErr: "duplicate outputFile",
		},
		// Unique explicit outputFile values are allowed.
		{
			name: "unique outputFiles are valid",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", OutputFile: "/out/c1.jsonl", Scope: "core"},
				{Name: "c2", OutputFile: "/out/c2.jsonl", Scope: "core"},
			}},
		},
		// Multi-cluster with scope=all and no acceptCRDs is rejected because
		// interactive CRD prompts can't be used concurrently.
		{
			name: "multi-cluster scope=all without acceptCRDs rejected",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", Scope: "all"},
				{Name: "c2", Scope: "all"},
			}},
			wantErr: "interactive CRD discovery is not supported in multi-cluster mode",
		},
		// Multi-cluster with empty scope (defaults to core, safe) is ok.
		{
			name: "multi-cluster empty scope (defaults to core) is ok",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", Scope: "core"},
				{Name: "c2", Scope: "core"},
			}},
		},
		// Multi-cluster with scope=all is ok when acceptCRDs is true in defaults.
		{
			name: "multi-cluster scope=all with default acceptCRDs=true is ok",
			cfg: Config{
				Defaults: ClusterDefaults{AcceptCRDs: true},
				Clusters: []ClusterEntry{
					{Name: "c1", Scope: "all"},
					{Name: "c2", Scope: "all"},
				},
			},
		},
		// Multi-cluster with scope=all is ok when per-cluster acceptCRDs is true.
		{
			name: "multi-cluster scope=all with per-cluster acceptCRDs=true is ok",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", Scope: "all", AcceptCRDs: boolPtr(true)},
				{Name: "c2", Scope: "all", AcceptCRDs: boolPtr(true)},
			}},
		},
		// Multi-cluster with empty scope and no acceptCRDs is rejected (empty
		// scope is treated as potentially interactive).
		{
			name: "multi-cluster empty scope without acceptCRDs rejected",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1"},
				{Name: "c2"},
			}},
			wantErr: "interactive CRD discovery is not supported in multi-cluster mode",
		},
		// Multi-cluster with a discoveryAllowlist is safe even without acceptCRDs.
		{
			name: "multi-cluster with discoveryAllowlist is ok without acceptCRDs",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", DiscoveryAllowlist: "./allowlist.txt"},
				{Name: "c2", DiscoveryAllowlist: "./allowlist.txt"},
			}},
		},
		// Single cluster is always ok regardless of scope/acceptCRDs.
		{
			name: "single cluster scope=all without acceptCRDs is ok",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", Scope: "all"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	t.Run("cluster inherits all defaults", func(t *testing.T) {
		cfg := &Config{
			Defaults: ClusterDefaults{
				Scope:         "core",
				AllNamespaces: true,
				Redacted:      true,
				AcceptCRDs:    false,
				Concurrency:   5,
				PaginateLimit: 50,
				ClusterType:   "kubernetes",
				OutputDir:     "/out",
			},
			Clusters: []ClusterEntry{{Name: "dev"}},
		}
		entries := ApplyDefaults(cfg)
		e := entries[0]
		if e.Scope != "core" {
			t.Errorf("scope: got %q", e.Scope)
		}
		if e.AllNamespaces == nil || !*e.AllNamespaces {
			t.Errorf("allNamespaces: expected true")
		}
		if e.Redacted == nil || !*e.Redacted {
			t.Errorf("redacted: expected true")
		}
		if e.Concurrency != 5 {
			t.Errorf("concurrency: got %d", e.Concurrency)
		}
		if e.PaginateLimit != 50 {
			t.Errorf("paginateLimit: got %d", e.PaginateLimit)
		}
		if e.ClusterType != "kubernetes" {
			t.Errorf("clusterType: got %q", e.ClusterType)
		}
		if e.OutputDir != "/out" {
			t.Errorf("outputDir: got %q", e.OutputDir)
		}
	})

	t.Run("cluster overrides all defaults", func(t *testing.T) {
		falseVal := false
		cfg := &Config{
			Defaults: ClusterDefaults{
				Scope:         "core",
				AllNamespaces: true,
				Redacted:      true,
				Concurrency:   5,
				PaginateLimit: 50,
				ClusterType:   "kubernetes",
				OutputDir:     "/default-out",
			},
			Clusters: []ClusterEntry{{
				Name:          "prod",
				Scope:         "allowlist",
				AllNamespaces: &falseVal,
				Namespace:     "prod,monitoring",
				Redacted:      &falseVal,
				Concurrency:   20,
				PaginateLimit: 200,
				ClusterType:   "openshift",
				OutputDir:     "/prod-out",
			}},
		}
		entries := ApplyDefaults(cfg)
		e := entries[0]
		if e.Scope != "allowlist" {
			t.Errorf("scope: got %q", e.Scope)
		}
		if e.AllNamespaces == nil || *e.AllNamespaces {
			t.Errorf("allNamespaces: expected false override")
		}
		if e.Namespace != "prod,monitoring" {
			t.Errorf("namespace: got %q", e.Namespace)
		}
		if e.Redacted == nil || *e.Redacted {
			t.Errorf("redacted: expected false override")
		}
		if e.Concurrency != 20 {
			t.Errorf("concurrency: got %d", e.Concurrency)
		}
		if e.ClusterType != "openshift" {
			t.Errorf("clusterType: got %q", e.ClusterType)
		}
		if e.OutputDir != "/prod-out" {
			t.Errorf("outputDir: got %q", e.OutputDir)
		}
	})
}

func TestExpandEnvVars(t *testing.T) {
	t.Run("expands env ref", func(t *testing.T) {
		t.Setenv("MY_TOKEN", "secretvalue")
		cfg := &Config{
			Clusters: []ClusterEntry{
				{Name: "c1", Token: "${MY_TOKEN}"},
			},
		}
		if err := ExpandEnvVars(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Clusters[0].Token != "secretvalue" {
			t.Errorf("expected expanded token, got %q", cfg.Clusters[0].Token)
		}
	})

	t.Run("error on unset env var", func(t *testing.T) {
		os.Unsetenv("UNSET_TOKEN_VAR")
		cfg := &Config{
			Clusters: []ClusterEntry{
				{Name: "c1", Token: "${UNSET_TOKEN_VAR}"},
			},
		}
		if err := ExpandEnvVars(cfg); err == nil {
			t.Fatal("expected error for unset env var")
		}
	})

	t.Run("literal token passes through", func(t *testing.T) {
		cfg := &Config{
			Clusters: []ClusterEntry{
				{Name: "c1", Token: "literaltoken"},
			},
		}
		if err := ExpandEnvVars(cfg); err != nil {
			t.Fatalf("unexpected error for literal token: %v", err)
		}
		if cfg.Clusters[0].Token != "literaltoken" {
			t.Errorf("expected token unchanged, got %q", cfg.Clusters[0].Token)
		}
	})

	t.Run("empty token skipped", func(t *testing.T) {
		cfg := &Config{
			Clusters: []ClusterEntry{{Name: "c1"}},
		}
		if err := ExpandEnvVars(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestLoadConfigRoundtrip(t *testing.T) {
	path := writeTempYAML(t, `
defaults:
  scope: core
  allNamespaces: true
  concurrency: 10
  paginateLimit: 100
  clusterType: auto
  outputDir: ./out

clusters:
  - name: prod-us-east
    kubeconfig: ~/.kube/prod
    clusterType: kubernetes
    scope: allowlist
    discoveryAllowlist: ./allowlists/prod.txt
    outputFile: prod-us-east.jsonl

  - name: staging
    kubeconfig: ~/.kube/staging
    redacted: true
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(cfg.Clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(cfg.Clusters))
	}
	if cfg.Clusters[0].Name != "prod-us-east" {
		t.Errorf("unexpected first cluster name: %q", cfg.Clusters[0].Name)
	}
	if cfg.Defaults.Concurrency != 10 {
		t.Errorf("unexpected default concurrency: %d", cfg.Defaults.Concurrency)
	}

	entries := ApplyDefaults(cfg)
	staging := entries[1]
	if staging.Scope != "core" {
		t.Errorf("staging scope should inherit default 'core', got %q", staging.Scope)
	}
	if staging.Concurrency != 10 {
		t.Errorf("staging concurrency should inherit 10, got %d", staging.Concurrency)
	}
	if staging.Redacted == nil || !*staging.Redacted {
		t.Errorf("staging redacted should be true (explicit override)")
	}
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "clusters.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp yaml: %v", err)
	}
	return path
}

// TestApplyDefaults_ClusterConcurrency verifies that ClusterConcurrency is
// accessible on the Config.Defaults after loading — it is a run-level knob
// and not merged per-cluster.
func TestApplyDefaults_ClusterConcurrency(t *testing.T) {
	path := writeTempYAML(t, `
defaults:
  clusterConcurrency: 4
clusters:
  - name: dev
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Defaults.ClusterConcurrency != 4 {
		t.Errorf("expected ClusterConcurrency 4, got %d", cfg.Defaults.ClusterConcurrency)
	}
}

// TestValidate_DuplicateOutputFile is a focused test for the duplicate
// outputFile check added as part of the concurrent multi-cluster feature.
func TestValidate_DuplicateOutputFile(t *testing.T) {
	cfg := &Config{
		Clusters: []ClusterEntry{
			{Name: "c1", OutputFile: "/shared/out.jsonl", Scope: "core"},
			{Name: "c2", OutputFile: "/shared/out.jsonl", Scope: "core"},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for duplicate outputFile, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate outputFile") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestValidate_InteractiveCRDs verifies the CRD validation cases in detail.
func TestValidate_InteractiveCRDs(t *testing.T) {
	truePtrs := func() *bool { b := true; return &b }
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "two clusters with scope=all and no controls — rejected",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "a", Scope: "all"},
				{Name: "b", Scope: "all"},
			}},
			wantErr: true,
		},
		{
			name: "two clusters with default scope (empty) — rejected",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "a"},
				{Name: "b"},
			}},
			wantErr: true,
		},
		{
			name: "two clusters scope=all with default acceptCRDs — ok",
			cfg: Config{
				Defaults: ClusterDefaults{AcceptCRDs: true},
				Clusters: []ClusterEntry{
					{Name: "a", Scope: "all"},
					{Name: "b", Scope: "all"},
				},
			},
			wantErr: false,
		},
		{
			name: "two clusters scope=all with per-cluster acceptCRDs — ok",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "a", Scope: "all", AcceptCRDs: truePtrs()},
				{Name: "b", Scope: "all", AcceptCRDs: truePtrs()},
			}},
			wantErr: false,
		},
		{
			name: "two clusters with default discoveryAllowlist — ok",
			cfg: Config{
				Defaults: ClusterDefaults{DiscoveryAllowlist: "./allowlist.txt"},
				Clusters: []ClusterEntry{
					{Name: "a"},
					{Name: "b"},
				},
			},
			wantErr: false,
		},
		{
			name: "two clusters with scope=core — ok (safe scope)",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "a", Scope: "core"},
				{Name: "b", Scope: "core"},
			}},
			wantErr: false,
		},
		{
			name: "single cluster with scope=all — ok (no multi-cluster restriction)",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "a", Scope: "all"},
			}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.cfg)
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), "interactive CRD discovery") {
				t.Errorf("expected CRD-related error, got: %v", err)
			}
		})
	}
}

func writeTestCA(t *testing.T) string {
	t.Helper()
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "t"}, NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true}
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(p, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestApplyDefaults_ParseUndefinedNodes(t *testing.T) {
	f := false
	cfg := &Config{
		Defaults: ClusterDefaults{ParseUndefinedNodes: true},
		Clusters: []ClusterEntry{{Name: "inherit"}, {Name: "override", ParseUndefinedNodes: &f}},
	}
	got := ApplyDefaults(cfg)
	if !*got[0].ParseUndefinedNodes || *got[1].ParseUndefinedNodes {
		t.Errorf("want inherit=true override=false, got %v %v", *got[0].ParseUndefinedNodes, *got[1].ParseUndefinedNodes)
	}
}

func TestInlineAllowlist_ReplaceSemantics(t *testing.T) {
	cfg := &Config{
		Defaults: ClusterDefaults{DiscoveryAllowlistEntries: []string{"apps/*"}},
		Clusters: []ClusterEntry{{Name: "inherit"}, {Name: "own", DiscoveryAllowlistEntries: []string{"batch/*"}}},
	}
	got := ApplyDefaults(cfg)
	if !reflect.DeepEqual(got[0].DiscoveryAllowlistEntries, []string{"apps/*"}) {
		t.Errorf("inherit: got %v", got[0].DiscoveryAllowlistEntries)
	}
	if !reflect.DeepEqual(got[1].DiscoveryAllowlistEntries, []string{"batch/*"}) {
		t.Errorf("own list must replace default, got %v", got[1].DiscoveryAllowlistEntries)
	}
}

func TestValidate_InlineAllowlist(t *testing.T) {
	bad := &Config{Clusters: []ClusterEntry{{Name: "a", DiscoveryAllowlistEntries: []string{"a/b/c/d"}}}}
	if err := Validate(bad); err == nil {
		t.Error("expected invalid inline entry to be rejected")
	}
	badDefault := &Config{Defaults: ClusterDefaults{DiscoveryAllowlistEntries: []string{"a/b/c/d"}}, Clusters: []ClusterEntry{{Name: "a"}}}
	if err := Validate(badDefault); err == nil {
		t.Error("expected invalid default inline entry to be rejected")
	}
}

// An inline allowlist satisfies the non-interactive CRD requirement.
func TestValidate_InlineAllowlistSatisfiesCRDGate(t *testing.T) {
	entries := []string{"cert-manager.io/*"}
	cfg := &Config{
		Defaults: ClusterDefaults{Scope: "all"},
		Clusters: []ClusterEntry{{Name: "a", DiscoveryAllowlistEntries: entries}, {Name: "b", DiscoveryAllowlistEntries: entries}},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected: %v", err)
	}
	cfg.Clusters[1].DiscoveryAllowlistEntries = nil
	if err := Validate(cfg); err == nil {
		t.Error("cluster without allowlist/acceptCRDs and scope all must still fail")
	}
	cfg.Defaults.DiscoveryAllowlistEntries = entries
	if err := Validate(cfg); err != nil {
		t.Errorf("default inline list should satisfy gate: %v", err)
	}
}

func TestValidate_TLS(t *testing.T) {
	ca := writeTestCA(t)
	tr := true
	notPEM := filepath.Join(t.TempDir(), "bad.pem")
	_ = os.WriteFile(notPEM, []byte("nope"), 0o600)
	tok := ClusterEntry{Name: "t", Server: "https://x", Token: "tok"}
	tests := []struct {
		name    string
		mutate  func(e *ClusterEntry)
		wantErr bool
	}{
		{"ca ok", func(e *ClusterEntry) { e.CACert = ca }, false},
		{"insecure ok", func(e *ClusterEntry) { e.InsecureSkipTLSVerify = &tr }, false},
		{"both", func(e *ClusterEntry) { e.CACert, e.InsecureSkipTLSVerify = ca, &tr }, true},
		{"missing file", func(e *ClusterEntry) { e.CACert = "/nonexistent/ca.pem" }, true},
		{"not pem", func(e *ClusterEntry) { e.CACert = notPEM }, true},
		{"kubeconfig cluster", func(e *ClusterEntry) { e.Server, e.Token, e.Kubeconfig, e.CACert = "", "", "~/.kube/c", ca }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tok
			tt.mutate(&e)
			err := Validate(&Config{Clusters: []ClusterEntry{e}})
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyDefaults_TLS(t *testing.T) {
	tr := true
	cfg := &Config{
		Defaults: ClusterDefaults{CACert: "/ca.pem"},
		Clusters: []ClusterEntry{
			{Name: "inherit", Server: "https://x", Token: "t"},
			{Name: "optout", Server: "https://x", Token: "t", InsecureSkipTLSVerify: &tr},
			{Name: "kube", Kubeconfig: "~/.kube/c"},
		},
	}
	got := ApplyDefaults(cfg)
	if got[0].CACert != "/ca.pem" {
		t.Errorf("token cluster should inherit default CA, got %q", got[0].CACert)
	}
	if got[1].CACert != "" || !*got[1].InsecureSkipTLSVerify {
		t.Errorf("explicit insecure must not inherit CA: %+v", got[1])
	}
	if got[2].CACert != "" {
		t.Errorf("kubeconfig cluster must not inherit CA, got %q", got[2].CACert)
	}
}
