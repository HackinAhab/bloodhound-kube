package collector

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"bloodhound-kube/internal/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCollectionPlanTargetsForTypesWithAliasesAndDedup(t *testing.T) {
	cfg := &CollectionsConfig{Collections: []ResourceCollection{
		{Name: "pods", Kind: "Pod", APIPath: "v1/pods", APIVersion: "v1", Plural: "pods", ResourceType: "pods", Namespaced: true, Enabled: true, SupportedClusters: []utils.ClusterType{utils.ClusterTypeKubernetes}},
		{Name: "services", Kind: "Service", APIPath: "v1/services", APIVersion: "v1", Plural: "services", ResourceType: "services", Namespaced: true, Enabled: true, ShortNames: []string{"svc"}, SupportedClusters: []utils.ClusterType{utils.ClusterTypeKubernetes}},
	}}

	plan := NewCollectionPlan(cfg)
	targets, err := plan.TargetsForTypes([]string{"Pod", "v1/services", "svc", "pods"})
	if err != nil {
		t.Fatalf("expected aliases to resolve, got error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 deduplicated targets, got %d", len(targets))
	}
}

func TestCollectionPlanTargetsForTypesRejectsUnknown(t *testing.T) {
	cfg := &CollectionsConfig{Collections: []ResourceCollection{{Name: "pods", Kind: "Pod", APIPath: "v1/pods", APIVersion: "v1", Plural: "pods", ResourceType: "pods", Namespaced: true, Enabled: true, SupportedClusters: []utils.ClusterType{utils.ClusterTypeKubernetes}}}}
	plan := NewCollectionPlan(cfg)

	_, err := plan.TargetsForTypes([]string{"does-not-exist"})
	if err == nil {
		t.Fatalf("expected error for unknown target")
	}
}

func TestBuildMetadataResourcesAndResourceKind(t *testing.T) {
	list := &metav1.PartialObjectMetadataList{Items: []metav1.PartialObjectMetadata{
		{TypeMeta: metav1.TypeMeta{Kind: "Pod"}, ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "ns1"}},
	}}
	resources := buildMetadataResources(list, "", "v1")
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if got := resourceKind(resources[0]); got != "pod" {
		t.Fatalf("expected resource kind pod, got %q", got)
	}
}

func TestCheckpointFailedJobRetryAndSaveLoad(t *testing.T) {
	cp := NewCheckpoint("id-1", "out.jsonl", utils.ClusterTypeKubernetes, &utils.ClusterInfo{Platform: "kubernetes"}, 2)
	cp.AddFailedJob("pods", "ns1", "first")
	cp.AddFailedJob("pods", "ns1", "second")

	if len(cp.FailedJobs) != 1 {
		t.Fatalf("expected one failed job entry, got %d", len(cp.FailedJobs))
	}
	if cp.FailedJobs[0].Retries != 1 {
		t.Fatalf("expected retries=1, got %d", cp.FailedJobs[0].Retries)
	}

	cp.AddCompletedJob("pods", "ns1", 3, 2*time.Second)
	if !cp.IsJobCompleted("pods", "ns1") {
		t.Fatalf("expected job marked as completed")
	}

	checkpointPath := filepath.Join(t.TempDir(), ".run.checkpoint.json")
	if err := cp.Save(checkpointPath); err != nil {
		t.Fatalf("failed to save checkpoint: %v", err)
	}
	loaded, err := LoadCheckpoint(checkpointPath)
	if err != nil {
		t.Fatalf("failed to load checkpoint: %v", err)
	}
	if loaded.CollectionID != "id-1" {
		t.Fatalf("expected collection id id-1, got %q", loaded.CollectionID)
	}
	if !CheckpointExists(checkpointPath) {
		t.Fatalf("expected checkpoint file to exist")
	}
	if err := RemoveCheckpoint(checkpointPath); err != nil {
		t.Fatalf("failed removing checkpoint: %v", err)
	}
	if _, err := os.Stat(checkpointPath); !os.IsNotExist(err) {
		t.Fatalf("expected checkpoint file removed, stat err=%v", err)
	}
}
