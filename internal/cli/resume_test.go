package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func seedResumeCheckpoint(t *testing.T) (*collector.Checkpoint, string) {
	t.Helper()
	dir := t.TempDir()
	cp := collector.NewCheckpoint("saved-run", filepath.Join(dir, "existing.jsonl"), utils.ClusterTypeKubernetes, &utils.ClusterInfo{Platform: "kubernetes"}, 2)
	cp.APIServer = "https://saved.example"
	cp.Targets = []collector.CollectionTarget{{Name: "secrets", Resource: "secrets", Version: "v1", Kind: "Secret", Namespaced: true, FetchMode: collector.FetchModeFull}}
	cp.Namespaces = []string{"original", "second"}
	saved := CollectRequest{Output: cp.OutputFile, Namespaces: "original,second", ResourceTypes: []string{"secrets"}, Kubeconfig: "/saved/config", Context: "saved-context", ClusterType: "kubernetes", Scope: "all", Redacted: true, FetchModeFull: true, Concurrency: 7, PaginateLimit: 42}
	cp.Settings, _ = json.Marshal(saved)
	cp.Pipeline, _ = json.Marshal(PipelineRequest{ParseEnabled: false, ClusterName: "production", ZipOutput: true, ParseUndefinedNodes: true, ParsedOutputPath: filepath.Join(dir, "parsed.json")})
	if err := os.WriteFile(cp.OutputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".saved.checkpoint.json")
	if err := cp.Save(path); err != nil {
		t.Fatal(err)
	}
	return cp, path
}

func TestRestoreResumeSettingsAndExplicitTuning(t *testing.T) {
	cp, path := seedResumeCheckpoint(t)
	incoming := PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: path, ClusterType: "auto", Scope: "core", Concurrency: 10, PaginateLimit: 100}, ParseEnabled: true, ClusterName: "default"}
	res, err := restorePipelineRequest(incoming)
	if err != nil {
		t.Fatal(err)
	}
	if res.ParseEnabled || !res.ZipOutput || !res.ParseUndefinedNodes || res.ClusterName != "production" || res.ParsedOutputPath == "" {
		t.Fatalf("pipeline defaults replaced saved settings: %+v", res)
	}
	got, err := restoreCollectRequest(res.Collect, cp)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Redacted || !got.FetchModeFull || got.Scope != "all" || got.Namespaces != "original,second" || got.Context != "saved-context" || got.Concurrency != 7 || got.PaginateLimit != 42 {
		t.Fatalf("collection defaults replaced saved settings: %+v", got)
	}
	incoming.Collect.Concurrency, incoming.Collect.PaginateLimit = 20, 250
	incoming.Collect.ExplicitFlags = map[string]bool{"concurrency": true, "paginate-limit": true}
	got, err = restoreCollectRequest(incoming.Collect, cp)
	if err != nil || got.Concurrency != 20 || got.PaginateLimit != 250 {
		t.Fatalf("explicit tuning not restored: %+v, %v", got, err)
	}
}

func TestResumeRejectsConflictingFlags(t *testing.T) {
	_, path := seedResumeCheckpoint(t)
	for _, flag := range []string{"namespace", "all-namespaces", "output", "type", "redacted", "fetch-mode-full", "scope", "no-parse", "cluster", "zip", "parsed-output", "parse-undefined-nodes"} {
		t.Run(flag, func(t *testing.T) {
			req := PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: path, AllNamespaces: true, ExplicitFlags: map[string]bool{flag: true}}, ParseEnabled: true}
			_, err := restorePipelineRequest(req)
			if err == nil || !strings.Contains(err.Error(), "--"+flag) {
				t.Fatalf("expected conflicting --%s error, got %v", flag, err)
			}
		})
	}
}

func TestResumeRequiresOriginalOutputAndRejectsLegacy(t *testing.T) {
	cp, path := seedResumeCheckpoint(t)
	if err := os.Remove(cp.OutputFile); err != nil {
		t.Fatal(err)
	}
	_, err := resolveOutputAndCheckpoint(CollectRequest{Resume: true, CheckpointFile: path})
	if err == nil || !strings.Contains(err.Error(), "original output") {
		t.Fatalf("missing output accepted: %v", err)
	}
	cp.Version = "1.0"
	if err := cp.Save(path); err != nil {
		t.Fatal(err)
	}
	_, err = PipelineService{}.Run(context.Background(), PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: path}}, utils.New("error", true))
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("legacy checkpoint accepted: %v", err)
	}
}

func TestResumeRejectsClusterMismatchBeforeTouchingOutput(t *testing.T) {
	cp, path := seedResumeCheckpoint(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version" {
			t.Errorf("wrong cluster was collected: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"gitVersion":"v1.30.0"}`)
	}))
	defer server.Close()
	before := []byte("uncommitted output must remain untouched on invalid resume")
	if err := os.WriteFile(cp.OutputFile, before, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := (PipelineService{}).Run(context.Background(), PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: path, Server: server.URL, Token: "fresh"}, Out: io.Discard}, utils.New("error", true))
	if err == nil || !strings.Contains(err.Error(), "cluster mismatch") {
		t.Fatalf("wrong cluster accepted: %v", err)
	}
	after, err := os.ReadFile(cp.OutputFile)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("invalid resume modified output: %s, %v", after, err)
	}
}

func TestNewCollectionCannotOverwriteUnfinishedCheckpoint(t *testing.T) {
	cp, path := seedResumeCheckpoint(t)
	_, err := resolveOutputAndCheckpoint(CollectRequest{Output: cp.OutputFile, CheckpointFile: path})
	if err == nil || !strings.Contains(err.Error(), "--resume") {
		t.Fatalf("existing checkpoint could be overwritten: %v", err)
	}
}

func TestResumeTokenNotPersistedAndCanBeRefreshed(t *testing.T) {
	cp, path := seedResumeCheckpoint(t)
	var saved CollectRequest
	json.Unmarshal(cp.Settings, &saved)
	saved.Kubeconfig, saved.Server, saved.Token = "", cp.APIServer, "original-secret-token"
	cp.Settings, _ = json.Marshal(saved)
	if bytes.Contains(cp.Settings, []byte(saved.Token)) {
		t.Fatal("token persisted in checkpoint")
	}
	_, err := restoreCollectRequest(CollectRequest{Resume: true, CheckpointFile: path}, cp)
	if err == nil || !strings.Contains(err.Error(), "fresh --token") {
		t.Fatalf("expected fresh token requirement, got %v", err)
	}
	got, err := restoreCollectRequest(CollectRequest{Resume: true, CheckpointFile: path, Token: "refreshed-token"}, cp)
	if err != nil || got.Token != "refreshed-token" || got.Server != cp.APIServer {
		t.Fatalf("token refresh failed: %+v, %v", got, err)
	}
}

func TestNewCollectionSavesSettingsAndRetriesFailedJob(t *testing.T) {
	var retry atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if retry.Load() && r.URL.Path != "/version" && r.URL.Path != "/api/v1/namespaces/production/pods" {
			t.Errorf("retry repeated discovery: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		switch r.URL.Path {
		case "/version":
			io.WriteString(w, `{"gitVersion":"v1.30.0"}`)
		case "/api":
			io.WriteString(w, `{"kind":"APIVersions","versions":["v1"]}`)
		case "/apis":
			io.WriteString(w, `{"kind":"APIGroupList","groups":[]}`)
		case "/api/v1":
			io.WriteString(w, `{"kind":"APIResourceList","groupVersion":"v1","resources":[{"name":"pods","kind":"Pod","namespaced":true,"verbs":["list"]}]}`)
		case "/apis/apiextensions.k8s.io/v1/customresourcedefinitions":
			io.WriteString(w, `{"kind":"CustomResourceDefinitionList","apiVersion":"apiextensions.k8s.io/v1","items":[]}`)
		case "/api/v1/namespaces/production/pods":
			if !retry.Load() {
				w.WriteHeader(http.StatusForbidden)
				io.WriteString(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"Forbidden","code":403}`)
				return
			}
			io.WriteString(w, `{"kind":"PodList","apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"p","namespace":"production"}}]}`)
		default:
			t.Errorf("unexpected API call: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	req := PipelineRequest{Collect: CollectRequest{Server: server.URL, Token: "must-not-be-persisted", ClusterType: "kubernetes", Namespaces: "production", NamespaceFlagSet: true, ResourceTypes: []string{"pods"}, Output: filepath.Join(dir, "fresh.jsonl"), Scope: "all", Redacted: true, FetchModeFull: true, Concurrency: 3, PaginateLimit: 61}, ParseEnabled: false, ClusterName: "my-cluster", ZipOutput: true, Out: io.Discard}
	_, err := (PipelineService{}).Run(context.Background(), req, utils.New("error", true))
	var partial *PartialCollectionError
	if !errors.As(err, &partial) {
		t.Fatalf("failed collection not reported: %v", err)
	}
	path := filepath.Join(dir, ".fresh.checkpoint.json")
	cp, err := collector.LoadCheckpoint(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cp.ValidateResume(); err != nil {
		t.Fatalf("new checkpoint cannot be resumed: %v", err)
	}
	data, _ := os.ReadFile(path)
	if bytes.Contains(data, []byte(req.Collect.Token)) || cp.Namespaces[0] != "production" || len(cp.Targets) != 1 || len(cp.FailedJobs) != 1 || cp.OutputOffset != 0 {
		t.Fatalf("incorrect new checkpoint: %s", data)
	}
	retry.Store(true)
	resp, err := (PipelineService{}).Run(context.Background(), PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: path, Token: "replacement"}, ParseEnabled: true, Out: io.Discard}, utils.New("error", true))
	if err != nil || resp.ParsedPath != "" || resp.JSONLPath != req.Collect.Output {
		t.Fatalf("new run did not resume saved settings: %+v, %v", resp, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("retry left completed checkpoint: %v", err)
	}
}

func TestResumeInterruptedCollectionRestoresPlanAndDropsUncommittedTail(t *testing.T) {
	cp, path := seedResumeCheckpoint(t)
	var resumed atomic.Bool
	var originalLists, secondLists atomic.Int32
	secondStarted := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/version":
			io.WriteString(w, `{"gitVersion":"v1.30.0"}`)
		case "/api/v1/namespaces/original/secrets", "/api/v1/namespaces/second/secrets":
			ns := "original"
			if strings.Contains(r.URL.Path, "/second/") {
				ns = "second"
				secondLists.Add(1)
				if !resumed.Load() {
					secondStarted <- struct{}{}
					<-r.Context().Done()
					return
				}
			} else {
				originalLists.Add(1)
			}
			if r.URL.Query().Get("limit") != "42" {
				t.Errorf("saved pagination not restored: %s", r.URL.RawQuery)
			}
			json.NewEncoder(w).Encode(map[string]any{"apiVersion": "v1", "kind": "SecretList", "items": []any{map[string]any{"apiVersion": "v1", "kind": "Secret", "metadata": map[string]any{"name": "s", "namespace": ns}, "data": map[string]any{"password": "c2VjcmV0"}}}})
		default:
			t.Errorf("resume unexpectedly rediscovered/replanned: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	cp.APIServer = server.URL
	var settings CollectRequest
	json.Unmarshal(cp.Settings, &settings)
	settings.Server, settings.Kubeconfig, settings.Context, settings.Concurrency = server.URL, "", "", 1
	cp.Settings, _ = json.Marshal(settings)
	if err := cp.Save(path); err != nil {
		t.Fatal(err)
	}
	req := PipelineRequest{Collect: CollectRequest{Resume: true, CheckpointFile: path, Token: "fresh-token", Scope: "core", Concurrency: 10, PaginateLimit: 100}, ParseEnabled: true, Out: io.Discard}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := (PipelineService{}).Run(ctx, req, utils.New("error", true))
		done <- err
	}()
	select {
	case <-secondStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("second job did not start")
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		loaded, err := collector.LoadCheckpoint(path)
		if err == nil && len(loaded.CompletedJobs) == 1 {
			cp = loaded
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("first job was not checkpointed")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("interruption reported as success: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("collection did not stop")
	}
	// Simulate a crash after additional output was written but before its checkpoint.
	f, err := os.OpenFile(cp.OutputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("uncommitted partial JSON")
	f.Close()
	resumed.Store(true)
	resp, err := (PipelineService{}).Run(context.Background(), req, utils.New("error", true))
	if err != nil {
		t.Fatal(err)
	}
	if resp.JSONLPath != cp.OutputFile || resp.ParsedPath != "" {
		t.Fatalf("saved output/no-parse not restored: %+v", resp)
	}
	data, err := os.ReadFile(cp.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(data, []byte("\n")) != 2 || bytes.Contains(data, []byte("uncommitted")) || bytes.Contains(data, []byte("c2VjcmV0")) {
		t.Fatalf("duplicate, incomplete or unredacted output: %s", data)
	}
	if originalLists.Load() != 1 || secondLists.Load() != 2 {
		t.Fatalf("completed job was retried: original=%d second=%d", originalLists.Load(), secondLists.Load())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("successful checkpoint not removed: %v", err)
	}
}
