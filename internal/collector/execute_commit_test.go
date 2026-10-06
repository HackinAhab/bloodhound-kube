package collector

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"bloodhound-kube/internal/utils"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func TestCheckpointCommitFailuresPreserveRetryableProgress(t *testing.T) {
	for _, failure := range []string{"initial checkpoint", "output write", "checkpoint after output"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".run.checkpoint.json")
			var lists atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if failure == "checkpoint after output" {
					// Fail checkpoint persistence after initial checkpoint and output write.
					if err := os.Mkdir(path+".tmp", 0755); err != nil {
						t.Error(err)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"p","namespace":"ns"}}]}`))
			}))
			defer server.Close()
			dyn, err := dynamic.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			log := utils.New("error", true)
			c := &Collector{clients: &utils.Clients{Dynamic: dyn, ClusterType: utils.ClusterTypeKubernetes, ClusterInfo: &utils.ClusterInfo{Platform: "kubernetes"}}, logger: log}
			targets := []CollectionTarget{{Name: "pods", Version: "v1", Resource: "pods", Namespaced: true, FetchMode: FetchModeFull}}
			w, err := utils.NewAsyncWriter(dir, "run.jsonl", log)
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			if failure == "initial checkpoint" {
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
			}
			if failure == "output write" {
				if err := w.Close(); err != nil {
					t.Fatal(err)
				}
			}
			_, _, total, errs := RunCollectionWithCheckpoint(context.Background(), c, w, targets, []string{"ns"}, "run.jsonl", 1, log, nil, path)
			var commitErr *CheckpointCommitError
			if total != 0 || !errors.As(errors.Join(errs...), &commitErr) {
				t.Fatalf("failed commit reported as success: total=%d errors=%v", total, errs)
			}
			if failure == "initial checkpoint" {
				if lists.Load() != 0 {
					t.Fatal("jobs started before initial checkpoint was saved")
				}
				return
			}
			cp, err := LoadCheckpoint(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(cp.CompletedJobs) != 0 || cp.OutputOffset != 0 || cp.JobsRemaining != 1 {
				t.Fatalf("failed commit advanced saved progress: %+v", cp)
			}
			if failure == "checkpoint after output" {
				info, err := os.Stat(filepath.Join(dir, "run.jsonl"))
				if err != nil || info.Size() == 0 {
					t.Fatal("test did not produce an uncommitted output tail")
				}
			}
		})
	}
}
