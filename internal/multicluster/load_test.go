package multicluster

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			name: "missing name",
			cfg: Config{Clusters: []ClusterEntry{{}}},
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
			cfg: Config{Clusters: []ClusterEntry{{Name: "dev"}}},
		},
		{
			name: "valid server+token",
			cfg: Config{Clusters: []ClusterEntry{
				{Name: "c1", Server: "https://k8s", Token: "tok"},
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

