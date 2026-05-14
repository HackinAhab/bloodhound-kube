package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bloodhound-kube/internal/multicluster"
	"bloodhound-kube/internal/utils"
)

func TestRunMultiPipeline_ConfigLoadFailure(t *testing.T) {
	log := utils.New("error", true)
	req := PipelineRequest{ClustersConfigPath: "/nonexistent/clusters.yaml"}
	_, err := runMultiPipeline(context.Background(), req, log)
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestRunMultiPipeline_AllSucceed(t *testing.T) {
	path := writeMultiClusterYAML(t, `
clusters:
  - name: alpha
  - name: beta
`)
	calls := 0
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ utils.Logger) (PipelineResponse, error) {
		calls++
		return PipelineResponse{NodeCount: 10, EdgeCount: 5, Duration: time.Second}, nil
	}

	log := utils.New("error", true)
	resp, err := runMultiPipeline(context.Background(), PipelineRequest{ClustersConfigPath: path}, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 pipeline calls, got %d", calls)
	}
	if resp.NodeCount != 20 {
		t.Fatalf("expected 20 total nodes, got %d", resp.NodeCount)
	}
	if resp.EdgeCount != 10 {
		t.Fatalf("expected 10 total edges, got %d", resp.EdgeCount)
	}
}

func TestRunMultiPipeline_PartialFailure(t *testing.T) {
	path := writeMultiClusterYAML(t, `
clusters:
  - name: ok
  - name: broken
`)
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ utils.Logger) (PipelineResponse, error) {
		if req.ClusterName == "broken" {
			return PipelineResponse{}, errors.New("connection refused")
		}
		return PipelineResponse{NodeCount: 5}, nil
	}

	log := utils.New("error", true)
	resp, err := runMultiPipeline(context.Background(), PipelineRequest{ClustersConfigPath: path}, log)
	if err == nil {
		t.Fatal("expected error for partial failure")
	}
	if !strings.Contains(err.Error(), "1 of 2 clusters failed") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if resp.NodeCount != 5 {
		t.Fatalf("expected 5 nodes from successful cluster, got %d", resp.NodeCount)
	}
}

func TestRunMultiPipeline_GlobalFlagsPassThrough(t *testing.T) {
	path := writeMultiClusterYAML(t, `
clusters:
  - name: c1
`)
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })
	var captured PipelineRequest
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ utils.Logger) (PipelineResponse, error) {
		captured = req
		return PipelineResponse{}, nil
	}

	log := utils.New("error", true)
	outer := PipelineRequest{
		ClustersConfigPath: path,
		Collect: CollectRequest{
			Resume:        true,
			CheckpointFile: "/tmp/ckpt",
			FetchModeFull: true,
		},
	}
	_, _ = runMultiPipeline(context.Background(), outer, log)

	if !captured.Collect.Resume {
		t.Error("Resume flag not passed through")
	}
	if captured.Collect.CheckpointFile != "/tmp/ckpt" {
		t.Errorf("CheckpointFile not passed through: %q", captured.Collect.CheckpointFile)
	}
	if !captured.Collect.FetchModeFull {
		t.Error("FetchModeFull not passed through")
	}
}

func TestRunMultiPipeline_NoParsePassThrough(t *testing.T) {
	path := writeMultiClusterYAML(t, `
clusters:
  - name: c1
`)
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })
	var captured PipelineRequest
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ utils.Logger) (PipelineResponse, error) {
		captured = req
		return PipelineResponse{}, nil
	}

	log := utils.New("error", true)
	_, _ = runMultiPipeline(context.Background(), PipelineRequest{
		ClustersConfigPath: path,
		ParseEnabled:       false,
	}, log)

	if captured.ParseEnabled {
		t.Error("ParseEnabled should be false when --no-parse is set")
	}
}

func TestRunMultiPipeline_ParseFailRetainsJSONL(t *testing.T) {
	path := writeMultiClusterYAML(t, `
clusters:
  - name: c1
`)
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ utils.Logger) (PipelineResponse, error) {
		return PipelineResponse{JSONLPath: "/tmp/c1.jsonl"}, errors.New("parse error")
	}

	log := utils.New("error", true)
	_, err := runMultiPipeline(context.Background(), PipelineRequest{ClustersConfigPath: path}, log)
	if err == nil {
		t.Fatal("expected error for failed cluster")
	}
	if !strings.Contains(err.Error(), "1 of 1 clusters failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildClusterPipelineRequest_OutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	entry := testEntry("mycluster")
	entry.OutputDir = tmpDir

	outer := PipelineRequest{ParseEnabled: true}
	req, err := buildClusterPipelineRequest(entry, outer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(req.Collect.Output, "mycluster") {
		t.Errorf("expected cluster name in output path, got %q", req.Collect.Output)
	}
	if !strings.HasSuffix(req.Collect.Output, ".jsonl") {
		t.Errorf("expected .jsonl suffix, got %q", req.Collect.Output)
	}
	if req.ParsedOutputPath == "" {
		t.Error("expected non-empty ParsedOutputPath when ParseEnabled=true")
	}
	if !strings.HasSuffix(req.ParsedOutputPath, ".json") {
		t.Errorf("expected .json suffix for parsed path, got %q", req.ParsedOutputPath)
	}
}

func TestBuildClusterPipelineRequest_ExplicitOutputFile(t *testing.T) {
	explicitPath := filepath.Join(t.TempDir(), "explicit.jsonl")
	entry := testEntry("c1")
	entry.OutputFile = explicitPath

	req, err := buildClusterPipelineRequest(entry, PipelineRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Collect.Output != explicitPath {
		t.Errorf("expected explicit output path, got %q", req.Collect.Output)
	}
}

func TestResolveClusterOutputPath_FallsBackToOuterDir(t *testing.T) {
	tmpDir := t.TempDir()
	entry := testEntry("mycluster")
	outer := PipelineRequest{Collect: CollectRequest{Output: tmpDir}}

	got, err := resolveClusterOutputPath(entry, outer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Dir(got) != tmpDir {
		t.Errorf("expected output in %q, got %q", tmpDir, got)
	}
}

func TestClusterOutputDir(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "."},
		{".", "."},
		{"..", ".."},
		{"/out/", "/out/"},
		{"/out/file.jsonl", "/out"},
		{"/out/dir", "/out/dir"},
	}
	for _, c := range cases {
		got := clusterOutputDir(c.in)
		if got != c.want {
			t.Errorf("clusterOutputDir(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func writeMultiClusterYAML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "clusters.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write clusters yaml: %v", err)
	}
	return path
}

func testEntry(name string) multicluster.ClusterEntry {
	return multicluster.ClusterEntry{Name: name}
}
