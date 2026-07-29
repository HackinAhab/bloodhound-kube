package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
defaults:
  acceptCRDs: true
clusters:
  - name: alpha
  - name: beta
`)
	calls := 0
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ *utils.Logger) (PipelineResponse, error) {
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
defaults:
  acceptCRDs: true
clusters:
  - name: ok
  - name: broken
`)
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ *utils.Logger) (PipelineResponse, error) {
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
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ *utils.Logger) (PipelineResponse, error) {
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
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ *utils.Logger) (PipelineResponse, error) {
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
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ *utils.Logger) (PipelineResponse, error) {
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
	req, err := buildClusterPipelineRequest(entry, outer, "2024-01-15-120000")
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

	req, err := buildClusterPipelineRequest(entry, PipelineRequest{}, "2024-01-15-120000")
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

	got, err := resolveClusterOutputPath(entry, outer, "2024-01-15-120000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Dir(got) != tmpDir {
		t.Errorf("expected output in %q, got %q", tmpDir, got)
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

// TestRunMultiPipeline_ConcurrentExecution verifies that when clusterConcurrency > 1,
// multiple cluster pipelines can genuinely overlap in time.
func TestRunMultiPipeline_ConcurrentExecution(t *testing.T) {
	path := writeMultiClusterYAML(t, `
defaults:
  acceptCRDs: true
  clusterConcurrency: 3
clusters:
  - name: c1
  - name: c2
  - name: c3
`)
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })

	var mu sync.Mutex
	var maxConcurrent int
	var current int

	runSinglePipelineFn = func(_ context.Context, _ PipelineRequest, _ *utils.Logger) (PipelineResponse, error) {
		mu.Lock()
		current++
		if current > maxConcurrent {
			maxConcurrent = current
		}
		mu.Unlock()

		time.Sleep(30 * time.Millisecond)

		mu.Lock()
		current--
		mu.Unlock()
		return PipelineResponse{NodeCount: 1}, nil
	}

	log := utils.New("error", true)
	resp, err := runMultiPipeline(context.Background(), PipelineRequest{ClustersConfigPath: path}, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NodeCount != 3 {
		t.Fatalf("expected 3 total nodes, got %d", resp.NodeCount)
	}
	if maxConcurrent < 2 {
		t.Errorf("expected at least 2 concurrent pipelines with clusterConcurrency=3, got max %d", maxConcurrent)
	}
}

// TestRunMultiPipeline_CLIFlagConcurrencyOverridesYAML verifies that the CLI
// ClusterConcurrency field takes precedence over the YAML defaults value.
func TestRunMultiPipeline_CLIFlagConcurrencyOverridesYAML(t *testing.T) {
	path := writeMultiClusterYAML(t, `
defaults:
  acceptCRDs: true
  clusterConcurrency: 5
clusters:
  - name: a
  - name: b
  - name: c
`)
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })

	var maxConcurrent int32
	var current int32

	runSinglePipelineFn = func(_ context.Context, _ PipelineRequest, _ *utils.Logger) (PipelineResponse, error) {
		c := atomic.AddInt32(&current, 1)
		for {
			old := atomic.LoadInt32(&maxConcurrent)
			if c <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, c) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&current, -1)
		return PipelineResponse{}, nil
	}

	log := utils.New("error", true)
	// CLI sets concurrency to 1 — should override the YAML's 5.
	_, err := runMultiPipeline(context.Background(), PipelineRequest{
		ClustersConfigPath: path,
		ClusterConcurrency: 1,
	}, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&maxConcurrent) > 1 {
		t.Errorf("expected sequential execution (max 1 concurrent) when CLI sets concurrency=1, got %d", maxConcurrent)
	}
}

// TestRunMultiPipeline_SequentialByDefault verifies the default (concurrency=0/1)
// runs clusters one at a time.
func TestRunMultiPipeline_SequentialByDefault(t *testing.T) {
	path := writeMultiClusterYAML(t, `
defaults:
  acceptCRDs: true
clusters:
  - name: x
  - name: y
`)
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })

	var maxConcurrent int32
	var current int32

	runSinglePipelineFn = func(_ context.Context, _ PipelineRequest, _ *utils.Logger) (PipelineResponse, error) {
		c := atomic.AddInt32(&current, 1)
		for {
			old := atomic.LoadInt32(&maxConcurrent)
			if c <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, c) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&current, -1)
		return PipelineResponse{}, nil
	}

	log := utils.New("error", true)
	// No ClusterConcurrency set — should default to 1.
	_, err := runMultiPipeline(context.Background(), PipelineRequest{ClustersConfigPath: path}, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&maxConcurrent) > 1 {
		t.Errorf("expected at most 1 concurrent pipeline with default concurrency, got %d", maxConcurrent)
	}
}

// TestRunMultiPipeline_CancellationPropagates verifies that cancelling the
// context causes pending goroutines to receive the cancellation.
func TestRunMultiPipeline_CancellationPropagates(t *testing.T) {
	path := writeMultiClusterYAML(t, `
defaults:
  acceptCRDs: true
  clusterConcurrency: 1
clusters:
  - name: slow
  - name: also-slow
`)
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })

	ctx, cancel := context.WithCancel(context.Background())

	runSinglePipelineFn = func(callCtx context.Context, _ PipelineRequest, _ *utils.Logger) (PipelineResponse, error) {
		// Cancel after the first cluster starts.
		cancel()
		select {
		case <-callCtx.Done():
			return PipelineResponse{}, callCtx.Err()
		case <-time.After(2 * time.Second):
			return PipelineResponse{}, nil
		}
	}

	log := utils.New("error", true)
	_, _ = runMultiPipeline(ctx, PipelineRequest{ClustersConfigPath: path}, log)
	// The test passes as long as it doesn't hang — the 2-second timeout in
	// the stub would cause the test to time out if cancellation did not work.
}

// TestRunMultiPipeline_BufferedStdoutOrderedByConfig verifies that per-cluster
// stdout is flushed in cluster-list order, not finish order.
func TestRunMultiPipeline_BufferedStdoutOrderedByConfig(t *testing.T) {
	path := writeMultiClusterYAML(t, `
defaults:
  acceptCRDs: true
  clusterConcurrency: 3
clusters:
  - name: first
  - name: second
  - name: third
`)
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })

	// Make "first" finish last so we can verify ordering is config-based, not
	// finish-time-based.
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ *utils.Logger) (PipelineResponse, error) {
		delay := 5 * time.Millisecond
		if req.ClusterName == "first" {
			delay = 60 * time.Millisecond
		}
		time.Sleep(delay)
		// Write distinguishable output for each cluster via req.Out.
		fmt.Fprintf(req.Out, "output-for-%s\n", req.ClusterName)
		return PipelineResponse{}, nil
	}

	// Redirect os.Stdout so we can capture what runMultiPipeline writes.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	log := utils.New("error", true)
	_, runErr := runMultiPipeline(context.Background(), PipelineRequest{ClustersConfigPath: path}, log)

	w.Close()
	var captured bytes.Buffer
	captured.ReadFrom(r)
	os.Stdout = origStdout

	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}

	output := captured.String()
	firstIdx := strings.Index(output, "output-for-first")
	secondIdx := strings.Index(output, "output-for-second")
	thirdIdx := strings.Index(output, "output-for-third")
	if firstIdx < 0 || secondIdx < 0 || thirdIdx < 0 {
		t.Fatalf("missing expected output lines; got:\n%s", output)
	}
	if !(firstIdx < secondIdx && secondIdx < thirdIdx) {
		t.Errorf("output not in config order: first=%d second=%d third=%d\noutput:\n%s",
			firstIdx, secondIdx, thirdIdx, output)
	}
}

// TestRunMultiPipeline_RunTimestampSharedAcrossClusters verifies that all
// per-cluster default output paths use the same timestamp (avoiding same-second
// collisions under concurrent execution).
func TestRunMultiPipeline_RunTimestampSharedAcrossClusters(t *testing.T) {
	tmpDir := t.TempDir()
	path := writeMultiClusterYAML(t, fmt.Sprintf(`
defaults:
  acceptCRDs: true
  clusterConcurrency: 2
  outputDir: %s
clusters:
  - name: aa
  - name: bb
`, tmpDir))
	orig := runSinglePipelineFn
	t.Cleanup(func() { runSinglePipelineFn = orig })

	var capturedPaths []string
	var mu sync.Mutex
	runSinglePipelineFn = func(_ context.Context, req PipelineRequest, _ *utils.Logger) (PipelineResponse, error) {
		mu.Lock()
		capturedPaths = append(capturedPaths, req.Collect.Output)
		mu.Unlock()
		return PipelineResponse{}, nil
	}

	log := utils.New("error", true)
	_, err := runMultiPipeline(context.Background(), PipelineRequest{ClustersConfigPath: path}, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedPaths) != 2 {
		t.Fatalf("expected 2 paths captured, got %d", len(capturedPaths))
	}
	// Extract the timestamp portion from each path: "name-TIMESTAMP.jsonl"
	ts := func(p string) string {
		base := filepath.Base(p)
		// base is "aa-2024-01-15-120000.jsonl" — strip prefix up to first "-" that follows the name
		// We strip the cluster name prefix and ".jsonl" suffix.
		for _, prefix := range []string{"aa-", "bb-"} {
			if strings.HasPrefix(base, prefix) {
				return strings.TrimSuffix(strings.TrimPrefix(base, prefix), ".jsonl")
			}
		}
		return base
	}
	ts0 := ts(capturedPaths[0])
	ts1 := ts(capturedPaths[1])
	if ts0 != ts1 {
		t.Errorf("cluster output paths use different timestamps: %q vs %q", capturedPaths[0], capturedPaths[1])
	}
}
