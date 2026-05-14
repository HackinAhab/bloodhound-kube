package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bloodhound-kube/internal/collector"
	"bloodhound-kube/internal/utils"
)

func TestCollectServiceRunValidation(t *testing.T) {
	service := CollectService{}
	log := utils.New("error", true)

	tests := []struct {
		name    string
		req     CollectRequest
		errWant string
	}{
		{
			name: "all namespaces conflicts with namespace flag",
			req: CollectRequest{
				AllNamespaces:    true,
				NamespaceFlagSet: true,
			},
			errWant: "cannot use -A",
		},
		{
			name: "server without token",
			req: CollectRequest{
				Server: "https://example",
			},
			errWant: "--server and --token",
		},
		{
			name: "token without server",
			req: CollectRequest{
				Token: "abc",
			},
			errWant: "--server and --token",
		},
		{
			name: "invalid cluster type",
			req: CollectRequest{
				ClusterType: "kind-of",
			},
			errWant: "invalid cluster type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Run(context.Background(), tt.req, log)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errWant)
			}
			if !strings.Contains(err.Error(), tt.errWant) {
				t.Fatalf("expected error containing %q, got %q", tt.errWant, err.Error())
			}
		})
	}
}

func TestResolveClusterType(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{in: "", wantErr: false},
		{in: "auto", wantErr: false},
		{in: "kubernetes", wantErr: false},
		{in: "k8s", wantErr: false},
		{in: "openshift", wantErr: false},
		{in: "ocp", wantErr: false},
		{in: "bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			_, err := resolveClusterType(tt.in)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for %q", tt.in)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.in, err)
			}
		})
	}
}

func TestShouldIncludeCRDsNonInteractive(t *testing.T) {
	resources := []collector.DiscoveryResource{{IsCRD: true}, {IsCRD: false}}
	if !isInteractive() {
		included, err := shouldIncludeCRDs(CollectRequest{}, resources, utils.New("error", true))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if included {
			t.Fatalf("expected CRDs to be excluded in non-interactive mode")
		}
	}

	included, err := shouldIncludeCRDs(CollectRequest{DiscoveryAccept: true}, resources, utils.New("error", true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !included {
		t.Fatalf("expected CRDs to be included when discovery auto accept is set")
	}
}

func TestResolveOutputAndCheckpoint(t *testing.T) {
	res, err := resolveOutputAndCheckpoint(CollectRequest{Output: "custom.jsonl"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.filename != "custom.jsonl" {
		t.Fatalf("expected custom filename, got %q", res.filename)
	}
	if res.checkpointPath == "" {
		t.Fatalf("expected checkpoint path")
	}
}

func TestResolveOutputAndCheckpointResume(t *testing.T) {
	tmpDir := t.TempDir()
	checkpointPath := filepath.Join(tmpDir, ".resume.checkpoint.json")
	content := fmt.Sprintf(`{"version":"1.0","timestamp":"%s","cluster":{"type":"kubernetes","platform":"kubernetes"},"collection_id":"id","output_file":"existing.jsonl","completed_jobs":[],"failed_jobs":[],"total_jobs":1,"jobs_remaining":1}`,
		"2026-01-01T00:00:00Z")
	if err := os.WriteFile(checkpointPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to seed checkpoint: %v", err)
	}

	res, err := resolveOutputAndCheckpoint(CollectRequest{Resume: true, CheckpointFile: checkpointPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.filename != "existing.jsonl" {
		t.Fatalf("expected resume filename existing.jsonl, got %q", res.filename)
	}
}

func TestResolveCollectScope(t *testing.T) {
	tests := []struct {
		name    string
		req     CollectRequest
		want    collectScope
		wantErr bool
	}{
		{name: "default core", req: CollectRequest{}, want: collectScopeCore},
		{name: "explicit all", req: CollectRequest{Scope: "all"}, want: collectScopeAll},
		{name: "explicit allowlist with file", req: CollectRequest{Scope: "allowlist", DiscoveryAllowlist: "x"}, want: collectScopeAllowlist},
		{name: "legacy allowlist implies allowlist scope", req: CollectRequest{DiscoveryAllowlist: "x"}, want: collectScopeAllowlist},
		{name: "allowlist scope missing file", req: CollectRequest{Scope: "allowlist"}, wantErr: true},
		{name: "invalid scope", req: CollectRequest{Scope: "weird"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveCollectScope(tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestDecideCRDExclusionMode(t *testing.T) {
	tests := []struct {
		name        string
		includeCRDs bool
		want        crdExclusionMode
	}{
		{name: "include crds", includeCRDs: true, want: crdExclusionNone},
		{name: "strip when declined", includeCRDs: false, want: crdExclusionStrip},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decideCRDExclusionMode(tt.includeCRDs)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestApplyDiscoveryFilterByScopeRejectsEmpty(t *testing.T) {
	_, _, err := applyDiscoveryFilterByScope(nil, collectScopeAll, nil)
	if err == nil {
		t.Fatalf("expected error for empty resources")
	}
}

func TestParseOutputPath(t *testing.T) {
	dir, file := parseOutputPath("")
	if dir != "." {
		t.Fatalf("expected default dir '.', got %q", dir)
	}
	if !strings.HasSuffix(file, ".jsonl") {
		t.Fatalf("expected generated filename to end with .jsonl, got %q", file)
	}

	dir, file = parseOutputPath("/tmp/")
	if dir != "/tmp/" {
		t.Fatalf("expected /tmp/ dir, got %q", dir)
	}
	if !strings.HasSuffix(file, ".jsonl") {
		t.Fatalf("expected generated filename to end with .jsonl, got %q", file)
	}

	dir, file = parseOutputPath("custom.jsonl")
	if dir != "." || file != "custom.jsonl" {
		t.Fatalf("expected ./custom.jsonl, got dir=%q file=%q", dir, file)
	}

	dir, file = parseOutputPath("./tmp/pipeline")
	if dir != "tmp" || file != "pipeline.jsonl" {
		t.Fatalf("expected tmp/pipeline.jsonl, got dir=%q file=%q", dir, file)
	}

	dir, file = parseOutputPath("pipeline")
	if dir != "." || file != "pipeline.jsonl" {
		t.Fatalf("expected ./pipeline.jsonl, got dir=%q file=%q", dir, file)
	}
}
