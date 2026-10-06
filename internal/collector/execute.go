package collector

import (
	"bloodhound-kube/internal/utils"
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const defaultListPageSize = 500

type CollectionJob struct {
	Target    CollectionTarget
	Namespace string
}

type CollectionResult struct {
	JobType   string
	Namespace string
	Resources []map[string]any
	Duration  time.Duration
	Error     error
}

// CheckpointCommitError stops the pipeline when output/progress cannot be committed.
type CheckpointCommitError struct{ Err error }

func (e *CheckpointCommitError) Error() string {
	return "failed to commit collection progress: " + e.Err.Error()
}
func (e *CheckpointCommitError) Unwrap() error { return e.Err }

func RunCollectionWithCheckpoint(ctx context.Context, c *Collector, w *utils.AsyncWriter, targets []CollectionTarget, namespacesToCollect []string, filename string, concurrency int, log *utils.Logger, existingCheckpoint *Checkpoint, checkpointFile string) (time.Duration, map[string]int, int, []error) {
	startTime := time.Now()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	counts := make(map[string]int)
	totalCollected := 0
	errors := make([]error, 0)

	clusterJobs := make([]CollectionJob, 0)
	namespacedJobs := make([]CollectionJob, 0)
	for _, target := range targets {
		if target.ClusterScoped {
			clusterJobs = append(clusterJobs, CollectionJob{Target: target})
			continue
		}
		for _, namespace := range namespacesToCollect {
			namespacedJobs = append(namespacedJobs, CollectionJob{Target: target, Namespace: namespace})
		}
	}
	allJobs := append(clusterJobs, namespacedJobs...)

	var checkpoint *Checkpoint
	if existingCheckpoint != nil {
		checkpoint = existingCheckpoint
	} else {
		collectionID := generateCollectionID()
		checkpoint = NewCheckpoint(collectionID, filename, c.GetClusterType(), c.clients.ClusterInfo, len(allJobs))
		checkpoint.Targets = targets
		checkpoint.Namespaces = namespacesToCollect
	}
	if err := checkpoint.Save(checkpointFile); err != nil {
		return time.Since(startTime), counts, 0, []error{&CheckpointCommitError{err}}
	}

	pendingJobs := make([]CollectionJob, 0, len(allJobs))
	for _, job := range allJobs {
		if !checkpoint.IsJobCompleted(job.Target.Name, job.Namespace) {
			pendingJobs = append(pendingJobs, job)
		}
	}

	log.Info("Collection plan", "total_jobs", len(allJobs), "completed_jobs", len(checkpoint.CompletedJobs), "pending_jobs", len(pendingJobs))
	if len(pendingJobs) == 0 {
		if err := ctx.Err(); err != nil {
			return time.Since(startTime), counts, 0, []error{err}
		}
		if err := finishCollectionCheckpoint(checkpoint, checkpointFile); err != nil {
			return time.Since(startTime), counts, 0, []error{&CheckpointCommitError{err}}
		}
		return time.Since(startTime), counts, 0, nil
	}

	checkpoint.JobsRemaining = len(pendingJobs)
	if concurrency < 1 {
		concurrency = 4
		log.Warn("Invalid concurrency value, using default", "concurrency", concurrency)
	}

	jobs := make(chan CollectionJob, max(len(pendingJobs), 100))
	results := make(chan CollectionResult, 200)

	var wg sync.WaitGroup
	for range concurrency {
		wg.Go(func() {
			checkpointWorker(ctx, c, jobs, results, log)
		})
	}

	for _, job := range pendingJobs {
		jobs <- job
	}
	close(jobs)
	go func() {
		wg.Wait()
		close(results)
	}()
	commitFailed := false
	for result := range results {
		if commitFailed {
			continue // Drain workers after cancellation without advancing progress.
		}
		if result.Error != nil {
			checkpoint.AddFailedJob(result.JobType, result.Namespace, result.Error.Error())
			errors = append(errors, result.Error)
		} else {
			batch := make([]any, len(result.Resources))
			for i, resource := range result.Resources {
				batch[i] = resource
			}
			if err := w.WriteJSONLBatch(batch); err != nil {
				errors = append(errors, &CheckpointCommitError{err})
				commitFailed = true
				cancel()
				continue
			}
			offset, err := w.Commit()
			if err != nil {
				errors = append(errors, &CheckpointCommitError{err})
				commitFailed = true
				cancel()
				continue
			}
			checkpoint.OutputOffset = offset
			checkpoint.AddCompletedJob(result.JobType, result.Namespace, len(result.Resources), result.Duration)
		}
		if err := checkpoint.Save(checkpointFile); err != nil {
			errors = append(errors, &CheckpointCommitError{err})
			commitFailed = true
			cancel()
			continue
		}
		if result.Error == nil {
			for _, resource := range result.Resources {
				counts[resourceKind(resource)]++
				totalCollected++
			}
		}
	}
	if err := ctx.Err(); err != nil {
		errors = append(errors, err)
	}
	completed, total, _ := checkpoint.GetProgress()
	if len(errors) == 0 && completed == total {
		if err := finishCollectionCheckpoint(checkpoint, checkpointFile); err != nil {
			errors = append(errors, &CheckpointCommitError{err})
		}
	}

	return time.Since(startTime), counts, totalCollected, errors
}

func finishCollectionCheckpoint(cp *Checkpoint, path string) error {
	cp.Phase = "collected"
	if err := cp.Save(path); err != nil {
		return err
	}
	if !cp.Retain {
		return RemoveCheckpoint(path)
	}
	return nil
}

func checkpointWorker(ctx context.Context, c *Collector, jobs <-chan CollectionJob, results chan<- CollectionResult, log *utils.Logger) {
	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		started := time.Now()
		resources, err := collectTarget(ctx, c, job.Target, job.Namespace)
		result := CollectionResult{JobType: job.Target.Name, Namespace: job.Namespace, Resources: resources, Duration: time.Since(started), Error: err}
		if err != nil {
			log.Error("Failed to collect resources", "type", job.Target.Name, "namespace", job.Namespace, "duration", result.Duration, "error", err)
		}

		select {
		case results <- result:
		case <-ctx.Done():
			return
		}
	}
}

func collectTarget(ctx context.Context, c *Collector, target CollectionTarget, namespace string) ([]map[string]any, error) {
	gvr := schema.GroupVersionResource{Group: target.Group, Version: target.Version, Resource: target.Resource}
	apiVersion := target.Version
	if target.Group != "" {
		apiVersion = target.Group + "/" + target.Version
	}

	var resources []map[string]any
	continueToken := ""
	for {
		listOpts := metav1.ListOptions{Limit: int64(c.GetPaginateLimit(defaultListPageSize)), Continue: continueToken}
		if target.FetchMode == FetchModeMetadata {
			list, err := listMetadata(ctx, c, gvr, target.ClusterScoped, namespace, listOpts)
			if err != nil {
				return nil, err
			}
			resources = append(resources, buildMetadataResources(list, target.Kind, apiVersion)...)
			continueToken = list.GetContinue()
		} else {
			list, err := listDynamic(ctx, c, gvr, target.ClusterScoped, namespace, listOpts)
			if err != nil {
				return nil, err
			}
			resources = append(resources, buildDynamicResources(list, target.Resource, c.IsRedacted())...)
			continueToken = list.GetContinue()
		}
		if continueToken == "" {
			break
		}
	}
	return resources, nil
}

func listDynamic(ctx context.Context, c *Collector, gvr schema.GroupVersionResource, clusterScoped bool, namespace string, listOpts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	if clusterScoped {
		return c.clients.Dynamic.Resource(gvr).List(ctx, listOpts)
	}
	return c.clients.Dynamic.Resource(gvr).Namespace(namespace).List(ctx, listOpts)
}

func listMetadata(ctx context.Context, c *Collector, gvr schema.GroupVersionResource, clusterScoped bool, namespace string, listOpts metav1.ListOptions) (*metav1.PartialObjectMetadataList, error) {
	if clusterScoped {
		return c.clients.Metadata.Resource(gvr).List(ctx, listOpts)
	}
	return c.clients.Metadata.Resource(gvr).Namespace(namespace).List(ctx, listOpts)
}

func buildDynamicResources(list *unstructured.UnstructuredList, resource string, redacted bool) []map[string]any {
	resources := make([]map[string]any, 0, len(list.Items))
	for _, item := range list.Items {
		processed := applyCollectionHelpers(item.Object, resource, redacted)
		if processed != nil {
			resources = append(resources, processed)
		}
	}
	return resources
}

func buildMetadataResources(list *metav1.PartialObjectMetadataList, fallbackKind, apiVersion string) []map[string]any {
	resources := make([]map[string]any, 0, len(list.Items))
	for _, item := range list.Items {
		kind := fallbackKind
		if kind == "" {
			kind = item.Kind
		}
		resources = append(resources, map[string]any{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]any{
				"name":            item.Name,
				"namespace":       item.Namespace,
				"uid":             string(item.UID),
				"resourceVersion": item.ResourceVersion,
				"labels":          mapStringToAny(item.Labels),
				"annotations":     mapStringToAny(item.Annotations),
			},
		})
	}
	return resources
}

func generateCollectionID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

func resourceKind(resource map[string]any) string {
	if kind, ok := resource["kind"].(string); ok && kind != "" {
		return strings.ToLower(kind)
	}
	return "unknown"
}
