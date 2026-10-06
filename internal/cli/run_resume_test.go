package cli

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"bloodhound-kube/internal/collector"
	"bloodhound-kube/internal/utils"
)

func testCollectionAPI(t *testing.T, list func(http.ResponseWriter, *http.Request), discovery *atomic.Int32) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/version":
			io.WriteString(w, `{"gitVersion":"v1.30.0"}`)
		case "/api":
			if discovery != nil {
				discovery.Add(1)
			}
			io.WriteString(w, `{"kind":"APIVersions","versions":["v1"]}`)
		case "/apis":
			io.WriteString(w, `{"kind":"APIGroupList","groups":[]}`)
		case "/api/v1":
			io.WriteString(w, `{"kind":"APIResourceList","groupVersion":"v1","resources":[{"name":"pods","kind":"Pod","namespaced":true,"verbs":["list"]}]}`)
		case "/apis/apiextensions.k8s.io/v1/customresourcedefinitions":
			io.WriteString(w, `{"kind":"CustomResourceDefinitionList","apiVersion":"apiextensions.k8s.io/v1","items":[]}`)
		case "/api/v1/namespaces/production/pods":
			list(w, r)
		default:
			t.Errorf("unexpected API request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func writeTestPods(w http.ResponseWriter) {
	io.WriteString(w, `{"kind":"PodList","apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"p","namespace":"production"}}]}`)
}
func writeTestForbidden(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
	io.WriteString(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"Forbidden","code":403}`)
}

func TestParentResumeSkipsCompletedResumesActiveAndStartsQueued(t *testing.T) {
	t.Setenv("FIRST_RUN_TOKEN", "first-secret-token")
	t.Setenv("SECOND_RUN_TOKEN", "second-secret-token")
	t.Setenv("THIRD_RUN_TOKEN", "third-secret-token")
	var firstCalls, secondCalls, thirdCalls atomic.Int32
	var resumed atomic.Bool
	secondStarted := make(chan struct{}, 1)
	first := testCollectionAPI(t, func(w http.ResponseWriter, r *http.Request) { firstCalls.Add(1); writeTestPods(w) }, nil)
	second := testCollectionAPI(t, func(w http.ResponseWriter, r *http.Request) {
		secondCalls.Add(1)
		if !resumed.Load() {
			secondStarted <- struct{}{}
			<-r.Context().Done()
			return
		}
		if r.Header.Get("Authorization") != "Bearer refreshed-second-token" {
			t.Errorf("token was not refreshed: %s", r.Header.Get("Authorization"))
		}
		writeTestPods(w)
	}, nil)
	third := testCollectionAPI(t, func(w http.ResponseWriter, r *http.Request) { thirdCalls.Add(1); writeTestPods(w) }, nil)
	config := writeMultiClusterYAML(t, fmt.Sprintf(`
defaults:
  namespace: production
  clusterType: kubernetes
  acceptCRDs: true
  clusterConcurrency: 1
clusters:
  - name: first
    server: %s
    token: ${FIRST_RUN_TOKEN}
  - name: second
    server: %s
    token: ${SECOND_RUN_TOKEN}
  - name: third
    server: %s
    token: ${THIRD_RUN_TOKEN}
`, first.URL, second.URL, third.URL))
	parentPath := filepath.Join(t.TempDir(), "run.checkpoint.json")
	var output bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := (PipelineService{}).Run(ctx, PipelineRequest{ClustersConfigPath: config, Collect: CollectRequest{CheckpointFile: parentPath, ResourceTypes: []string{"pods"}, FetchModeFull: true}, Out: &output}, utils.New("error", true))
		done <- err
	}()
	select {
	case <-secondStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("second cluster never started")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("interruption reported as success: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("multi-cluster cancellation hung")
	}
	run, err := loadMultiRun(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if run.Clusters[0].State != "complete" || run.Clusters[1].State != "failed" || run.Clusters[2].State != "queued" || thirdCalls.Load() != 0 {
		t.Fatalf("incorrect interrupted run state: %+v", run.Clusters)
	}
	if !strings.Contains(output.String(), "Run checkpoint:") || !strings.Contains(output.String(), "QUEUED") {
		t.Fatalf("run recovery information not printed: %s", output.String())
	}
	data, _ := os.ReadFile(parentPath)
	for _, secret := range []string{"first-secret-token", "second-secret-token", "third-secret-token"} {
		if bytes.Contains(data, []byte(secret)) {
			t.Fatal("parent checkpoint contains bearer token")
		}
	}
	for _, index := range []int{0, 2} {
		if _, err := os.Stat(run.Clusters[index].Request.Collect.CheckpointFile); !os.IsNotExist(err) {
			t.Fatalf("completed/queued child has unexpected checkpoint: %v", err)
		}
	}
	// Environment references work without the original YAML. Completed clusters
	// must not require credentials that have expired since the original run.
	t.Setenv("FIRST_RUN_TOKEN", "")
	t.Setenv("SECOND_RUN_TOKEN", "refreshed-second-token")
	if err := os.Remove(config); err != nil {
		t.Fatal(err)
	}
	resumed.Store(true)
	resp, err := (PipelineService{}).Run(context.Background(), PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: parentPath}, Out: io.Discard}, utils.New("error", true))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Phase != "complete" || firstCalls.Load() != 1 || secondCalls.Load() != 2 || thirdCalls.Load() != 1 {
		t.Fatalf("incorrect recovery: %+v, calls=%d/%d/%d", resp, firstCalls.Load(), secondCalls.Load(), thirdCalls.Load())
	}
	for _, c := range run.Clusters {
		data, err := os.ReadFile(c.Request.Collect.Output)
		if err != nil || bytes.Count(data, []byte("\n")) != 1 {
			t.Fatalf("output path changed or duplicate resources written: %s, %v", data, err)
		}
		if _, err := os.Stat(c.Request.Collect.CheckpointFile); !os.IsNotExist(err) {
			t.Fatalf("successful child checkpoint retained: %v", err)
		}
	}
	if _, err := os.Stat(parentPath); !os.IsNotExist(err) {
		t.Fatalf("successful parent checkpoint retained: %v", err)
	}
}

func TestParentRefreshesYAMLCredentialsWithoutChangingSavedPlan(t *testing.T) {
	var calls, discovery atomic.Int32
	server := testCollectionAPI(t, func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		expected := "Bearer initial-secret"
		if call == 2 {
			expected = "Bearer replacement-secret"
		}
		if call == 3 {
			expected = "Bearer latest-secret"
		}
		if r.Header.Get("Authorization") != expected {
			t.Errorf("unexpected credentials: %s", r.Header.Get("Authorization"))
		}
		if call < 3 {
			writeTestForbidden(w)
			return
		}
		writeTestPods(w)
	}, &discovery)
	config := writeMultiClusterYAML(t, fmt.Sprintf(`defaults:
  namespace: production
  clusterType: kubernetes
clusters:
  - name: prod
    server: %s
    token: initial-secret
`, server.URL))
	parentPath := filepath.Join(t.TempDir(), "run.json")
	_, err := (PipelineService{}).Run(context.Background(), PipelineRequest{ClustersConfigPath: config, Collect: CollectRequest{CheckpointFile: parentPath, ResourceTypes: []string{"pods"}, FetchModeFull: true}, Out: io.Discard}, utils.New("error", true))
	if err == nil {
		t.Fatal("expected incomplete initial run")
	}
	originalDiscovery := discovery.Load()
	credentials := writeMultiClusterYAML(t, fmt.Sprintf(`defaults:
  namespace: changed-namespace
  scope: changed-scope
  redacted: true
clusters:
  - name: prod
    server: %s
    token: replacement-secret
`, server.URL))
	if err := os.Remove(config); err != nil {
		t.Fatal(err)
	}
	_, err = (PipelineService{}).Run(context.Background(), PipelineRequest{ClustersConfigPath: credentials, Collect: CollectRequest{Resume: true, CheckpointFile: parentPath, Concurrency: 20, ExplicitFlags: map[string]bool{"concurrency": true}}, Out: io.Discard}, utils.New("error", true))
	if err == nil {
		t.Fatal("expected incomplete first retry")
	}
	run, err := loadMultiRun(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if run.CredentialsPath != credentials || run.Clusters[0].Request.Collect.Concurrency != 20 {
		t.Fatalf("credential reference/tuning not retained: %+v", run)
	}
	data, _ := os.ReadFile(credentials)
	data = bytes.ReplaceAll(data, []byte("replacement-secret"), []byte("latest-secret"))
	if err := os.WriteFile(credentials, data, 0600); err != nil {
		t.Fatal(err)
	}
	_, err = (PipelineService{}).Run(context.Background(), PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: parentPath}, Out: io.Discard}, utils.New("error", true))
	if err != nil {
		t.Fatal(err)
	}
	if discovery.Load() != originalDiscovery || calls.Load() != 3 {
		t.Fatalf("active cluster was replanned: discovery=%d calls=%d", discovery.Load(), calls.Load())
	}
}

func TestIndividuallyCompletedChildCanBeAcknowledgedByParent(t *testing.T) {
	var retry atomic.Bool
	var calls atomic.Int32
	server := testCollectionAPI(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !retry.Load() {
			writeTestForbidden(w)
			return
		}
		writeTestPods(w)
	}, nil)
	config := writeMultiClusterYAML(t, fmt.Sprintf(`defaults:
  namespace: production
  clusterType: kubernetes
clusters:
  - name: prod
    server: %s
    token: initial-secret
`, server.URL))
	parentPath := filepath.Join(t.TempDir(), "run.json")
	_, err := (PipelineService{}).Run(context.Background(), PipelineRequest{ClustersConfigPath: config, Collect: CollectRequest{CheckpointFile: parentPath, ResourceTypes: []string{"pods"}, FetchModeFull: true}, Out: io.Discard}, utils.New("error", true))
	if err == nil {
		t.Fatal("expected initial failure")
	}
	run, err := loadMultiRun(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	child := run.Clusters[0].Request.Collect.CheckpointFile
	retry.Store(true)
	_, err = (PipelineService{}).Run(context.Background(), PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: child, Token: "refreshed"}, Out: io.Discard}, utils.New("error", true))
	if err != nil {
		t.Fatal(err)
	}
	cp, err := collector.LoadCheckpoint(child)
	if err != nil || cp.Phase != "complete" {
		t.Fatalf("child completion marker removed before parent acknowledgement: %+v, %v", cp, err)
	}
	if err := os.Remove(config); err != nil {
		t.Fatal(err)
	}
	_, err = (PipelineService{}).Run(context.Background(), PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: parentPath}, Out: io.Discard}, utils.New("error", true))
	if err != nil || calls.Load() != 2 {
		t.Fatalf("parent did not reconcile completed child: calls=%d error=%v", calls.Load(), err)
	}
	if _, err := os.Stat(child); !os.IsNotExist(err) {
		t.Fatalf("acknowledged child was not cleaned up: %v", err)
	}
}

func TestPostCollectionPhasesResumeWithoutCredentialsOrKubernetes(t *testing.T) {
	for _, phase := range []string{"collected", "parsed"} {
		t.Run(phase, func(t *testing.T) {
			cp, path := seedResumeCheckpoint(t)
			data := []byte(`{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"original"}}` + "\n")
			if err := os.WriteFile(cp.OutputFile, data, 0644); err != nil {
				t.Fatal(err)
			}
			cp.OutputOffset = int64(len(data))
			cp.AddCompletedJob("secrets", "original", 1, time.Second)
			cp.AddCompletedJob("secrets", "second", 0, time.Second)
			cp.Phase = phase
			parsedPath := filepath.Join(filepath.Dir(path), "parsed.json")
			pipeline := PipelineRequest{ParseEnabled: true, ParsedOutputPath: parsedPath, ClusterName: "original-cluster", ZipOutput: phase == "parsed"}
			cp.Pipeline, _ = json.Marshal(pipeline)
			blocked := parsedPath + ".tmp"
			if phase == "parsed" {
				cp.Artifact = parsedPath
				if err := os.WriteFile(parsedPath, []byte(`{"already":"parsed"}`), 0644); err != nil {
					t.Fatal(err)
				}
				blocked = parsedZipPath(parsedPath)
			}
			if err := os.Mkdir(blocked, 0755); err != nil {
				t.Fatal(err)
			}
			if err := cp.Save(path); err != nil {
				t.Fatal(err)
			}
			req := PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: path}, Out: io.Discard}
			_, err := (PipelineService{}).Run(context.Background(), req, utils.New("error", true))
			if err == nil {
				t.Fatal("expected post-processing failure")
			}
			loaded, err := collector.LoadCheckpoint(path)
			if err != nil || loaded.Phase != phase {
				t.Fatalf("pipeline phase lost after failure: %+v, %v", loaded, err)
			}
			if err := os.Remove(blocked); err != nil {
				t.Fatal(err)
			}
			resp, err := (PipelineService{}).Run(context.Background(), req, utils.New("error", true))
			if err != nil || resp.Phase != "complete" {
				t.Fatalf("post-processing attempted to recollect: %+v, %v", resp, err)
			}
			if phase == "parsed" {
				archive, err := zip.OpenReader(resp.ParsedPath)
				if err != nil {
					t.Fatal(err)
				}
				defer archive.Close()
				if len(archive.File) != 1 {
					t.Fatal("invalid ZIP artifact")
				}
				r, err := archive.File[0].Open()
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(r)
				r.Close()
				if err != nil || string(data) != `{"already":"parsed"}` {
					t.Fatalf("ZIP-only resume repeated parsing: %s, %v", data, err)
				}
				if _, err := os.Stat(parsedPath); !os.IsNotExist(err) {
					t.Fatalf("parsed source not removed after durable completion: %v", err)
				}
			}
		})
	}
}

func TestParentRecoversParseAndZIPWithoutClusterCredentials(t *testing.T) {
	var calls atomic.Int32
	server := testCollectionAPI(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); writeTestPods(w) }, nil)
	t.Setenv("POSTPROCESS_TOKEN", "temporary-token")
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "raw.jsonl")
	parsedPath := filepath.Join(dir, "raw.json")
	blocked := parsedPath + ".tmp"
	if err := os.Mkdir(blocked, 0755); err != nil {
		t.Fatal(err)
	}
	config := writeMultiClusterYAML(t, fmt.Sprintf(`defaults:
  namespace: production
  clusterType: kubernetes
clusters:
  - name: prod
    outputFile: %s
    server: %s
    token: ${POSTPROCESS_TOKEN}
`, outputPath, server.URL))
	parentPath := filepath.Join(dir, "run.json")
	_, err := (PipelineService{}).Run(context.Background(), PipelineRequest{ClustersConfigPath: config, Collect: CollectRequest{CheckpointFile: parentPath, ResourceTypes: []string{"pods"}, FetchModeFull: true}, ParseEnabled: true, ZipOutput: true, Out: io.Discard}, utils.New("error", true))
	if err == nil {
		t.Fatal("expected parse failure")
	}
	run, err := loadMultiRun(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if run.Clusters[0].Phase != "collected" {
		t.Fatalf("completed collection was not checkpointed: %+v", run.Clusters[0])
	}
	t.Setenv("POSTPROCESS_TOKEN", "")
	if err := os.Remove(config); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	resp, err := (PipelineService{}).Run(context.Background(), PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: parentPath}, Out: io.Discard}, utils.New("error", true))
	if err != nil || resp.NodeCount == 0 || calls.Load() != 1 {
		t.Fatalf("post-processing recovery recollected resources: %+v, calls=%d error=%v", resp, calls.Load(), err)
	}
	archive, err := zip.OpenReader(parsedZipPath(parsedPath))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	r, err := archive.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	r.Close()
	if err != nil || !bytes.Contains(data, []byte("prod:")) {
		t.Fatalf("saved cluster metadata was not restored: %s, %v", data, err)
	}
}

func TestParentAcknowledgesDurableChildAfterParentWriteFailure(t *testing.T) {
	parentPath := filepath.Join(t.TempDir(), "run.json")
	var calls atomic.Int32
	server := testCollectionAPI(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if err := os.Mkdir(parentPath+".tmp", 0755); err != nil {
			t.Error(err)
		}
		writeTestPods(w)
	}, nil)
	config := writeMultiClusterYAML(t, fmt.Sprintf(`defaults:
  namespace: production
  clusterType: kubernetes
clusters:
  - name: prod
    server: %s
    token: transient-token
`, server.URL))
	_, err := (PipelineService{}).Run(context.Background(), PipelineRequest{ClustersConfigPath: config, Collect: CollectRequest{CheckpointFile: parentPath, ResourceTypes: []string{"pods"}, FetchModeFull: true}, Out: io.Discard}, utils.New("error", true))
	if err == nil || !strings.Contains(err.Error(), "persist run progress") {
		t.Fatalf("parent write failure ignored: %v", err)
	}
	run, err := loadMultiRun(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	childPath := run.Clusters[0].Request.Collect.CheckpointFile
	cp, err := collector.LoadCheckpoint(childPath)
	if err != nil || cp.Phase != "complete" || run.Clusters[0].State != "active" {
		t.Fatalf("durable completion marker lost: %+v, %v", cp, err)
	}
	if err := os.Remove(parentPath + ".tmp"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(config); err != nil {
		t.Fatal(err)
	}
	_, err = (PipelineService{}).Run(context.Background(), PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: parentPath}, Out: io.Discard}, utils.New("error", true))
	if err != nil || calls.Load() != 1 {
		t.Fatalf("parent write recovery required recollection: calls=%d error=%v", calls.Load(), err)
	}
	if _, err := os.Stat(childPath); !os.IsNotExist(err) {
		t.Fatalf("acknowledged marker not cleaned up: %v", err)
	}
}

func TestMissingChildIsNotTreatedAsSuccessfulOrOverwritten(t *testing.T) {
	config := writeMultiClusterYAML(t, `clusters:
  - name: prod
`)
	run, parentPath, err := createMultiRun(PipelineRequest{ClustersConfigPath: config})
	if err != nil {
		t.Fatal(err)
	}
	run.Clusters[0].State, run.Clusters[0].Phase = "failed", "collected"
	data := []byte("existing original output")
	if err := os.WriteFile(run.Clusters[0].Request.Collect.Output, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := run.save(parentPath); err != nil {
		t.Fatal(err)
	}
	_, err = (PipelineService{}).Run(context.Background(), PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: parentPath}, Out: io.Discard}, utils.New("error", true))
	if err == nil || !strings.Contains(err.Error(), "missing child checkpoint") {
		t.Fatalf("missing checkpoint accepted: %v", err)
	}
	after, err := os.ReadFile(run.Clusters[0].Request.Collect.Output)
	if err != nil || !bytes.Equal(data, after) {
		t.Fatal("output was overwritten without a child checkpoint")
	}
}
