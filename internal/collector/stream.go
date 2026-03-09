package collector

import (
	"bloodhound-kube/internal/utils"
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"
)

type CollectionJob struct {
	Handler   ResourceHandler
	Namespace string
}

func RunCollectionWithCheckpoint(ctx context.Context, c *Collector, w *utils.AsyncWriter, typesToCollect, namespacesToCollect []string, filename string, concurrency int, log utils.Logger, existingCheckpoint *Checkpoint, checkpointFile string) (time.Duration, map[string]int, int, []error) {
	startTime := time.Now()
	stats := NewCollectionStats()

	clusterJobs := make([]CollectionJob, 0)
	namespacedJobs := make([]CollectionJob, 0)

	// Track handlers we've already added to avoid duplicates from nicknames
	seenHandlers := make(map[ResourceHandler]bool)

	for _, resourceType := range typesToCollect {
		handler, err := DefaultRegistry.GetHandler(resourceType)
		if err != nil {
			log.Error("Unknown resource type", "type", resourceType, "error", err)
			continue
		}

		// Skip if we've already added this handler (nickname deduplication)
		if seenHandlers[handler] {
			log.Debug("Skipping duplicate handler", "type", resourceType, "handler", handler.GetName())
			continue
		}
		seenHandlers[handler] = true

		if handler.IsClusterScoped() {
			clusterJobs = append(clusterJobs, CollectionJob{Handler: handler, Namespace: ""})
		} else {
			for _, ns := range namespacesToCollect {
				namespacedJobs = append(namespacedJobs, CollectionJob{Handler: handler, Namespace: ns})
			}
		}
	}

	allJobs := append(clusterJobs, namespacedJobs...)

	var checkpoint *Checkpoint
	if existingCheckpoint != nil {
		checkpoint = existingCheckpoint
	} else {
		collectionID := generateCollectionID()
		checkpoint = NewCheckpoint(collectionID, filename, c.GetClusterType(), c.clients.ClusterInfo, len(allJobs))
	}

	pendingJobs := make([]CollectionJob, 0)
	for _, job := range allJobs {
		if !checkpoint.IsJobCompleted(job.Handler.GetName(), job.Namespace) {
			pendingJobs = append(pendingJobs, job)
		}
	}

	log.Info("Collection plan", "total_jobs", len(allJobs), "completed_jobs", len(checkpoint.CompletedJobs), "pending_jobs", len(pendingJobs))

	if len(pendingJobs) == 0 {
		log.Info("All jobs already completed")
		counts, totalCollected, errors := stats.GetStats()
		return time.Since(startTime), counts, totalCollected, errors
	}

	checkpoint.JobsRemaining = len(pendingJobs)

	// Validate concurrency value
	if concurrency < 1 {
		concurrency = 4 // sensible default
		log.Warn("Invalid concurrency value, using default", "concurrency", concurrency)
	}

	jobBufferSize := max(len(pendingJobs), 100)

	jobs := make(chan CollectionJob, jobBufferSize)
	results := make(chan CollectionResult, 200)

	var wg sync.WaitGroup

	// Spawn workers correctly
	for i := 0; i < concurrency; i++ {
		wg.Go(func() {
			checkpointWorker(ctx, c, jobs, results, log)
		})
	}

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		batchBuffer := make([]any, 0, 100)

		flushBatch := func() {
			if len(batchBuffer) > 0 {
				flushStart := time.Now()
				if err := w.WriteJSONLBatch(batchBuffer); err != nil {
					log.Error("Failed to write batch", "size", len(batchBuffer), "error", err)
					stats.AddError(err)
				} else {
					log.Trace("Flushed batch", "size", len(batchBuffer), "duration", time.Since(flushStart))
				}
				for _, item := range batchBuffer {
					if res, ok := item.(map[string]any); ok {
						stats.AddCount(resourceKind(res), 1)
					}
				}
				batchBuffer = batchBuffer[:0]
			}
		}

		checkpointTicker := time.NewTicker(5 * time.Second)
		flushTicker := time.NewTicker(50 * time.Millisecond)
		defer checkpointTicker.Stop()
		defer flushTicker.Stop()

		for {
			select {
			case result, ok := <-results:
				if !ok {
					flushBatch()
					return
				}

				if result.Error != nil {
					checkpoint.AddFailedJob(result.JobType, result.Namespace, result.Error.Error())
					stats.AddError(result.Error)
				} else {
					checkpoint.AddCompletedJob(result.JobType, result.Namespace, len(result.Resources), result.Duration)
					for _, resource := range result.Resources {
						batchBuffer = append(batchBuffer, resource)
						if len(batchBuffer) >= 100 {
							flushBatch()
						}
					}
				}

			case <-checkpointTicker.C:
				if err := checkpoint.Save(checkpointFile); err != nil {
					log.Error("Failed to save checkpoint", "error", err)
				}

			case <-flushTicker.C:
				flushBatch()
			}
		}
	}()

	for _, job := range pendingJobs {
		jobs <- job
	}

	close(jobs)
	wg.Wait()
	close(results)
	<-writerDone

	if err := w.Flush(); err != nil {
		log.Error("Failed final flush", "error", err)
	}

	if err := checkpoint.Save(checkpointFile); err != nil {
		log.Error("Failed to save final checkpoint", "error", err)
	}

	counts, totalCollected, errors := stats.GetStats()
	duration := time.Since(startTime)

	if len(errors) > 0 {
		log.Error("Collection completed with errors", "error_count", len(errors))
		for i, err := range errors {
			if i < 5 {
				log.Error("Collection error", "error", err)
			}
		}
		if len(errors) > 5 {
			log.Error("Additional errors suppressed", "count", len(errors)-5)
		}
	}

	completed, total, pct := checkpoint.GetProgress()
	log.Info("Final progress", "completed", completed, "total", total, "percentage", fmt.Sprintf("%.1f%%", pct))

	// Remove checkpoint file after successful collection completion
	if len(errors) == 0 && completed == total {
		if err := RemoveCheckpoint(checkpointFile); err != nil {
			log.Warn("Failed to remove checkpoint file after successful collection", "checkpoint_file", checkpointFile, "error", err)
		} else {
			log.Debug("Removed checkpoint file after successful collection", "checkpoint_file", checkpointFile)
		}
	}

	return duration, counts, totalCollected, errors
}

type CollectionResult struct {
	JobType   string
	Namespace string
	Resources []map[string]any
	Duration  time.Duration
	Error     error
}

func checkpointWorker(ctx context.Context, c *Collector, jobs <-chan CollectionJob, results chan<- CollectionResult, log utils.Logger) {
	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		startTime := time.Now()
		if job.Namespace != "" {
			log.Trace("Starting collection job", "type", job.Handler.GetName(), "namespace", job.Namespace)
		} else {
			log.Trace("Starting collection job", "type", job.Handler.GetName())
		}
		batch, err := job.Handler.Collect(ctx, c, job.Namespace)
		duration := time.Since(startTime)

		result := CollectionResult{
			JobType:   job.Handler.GetName(),
			Namespace: job.Namespace,
			Duration:  duration,
		}

		if err != nil {
			if job.Namespace != "" {
				log.Error("Failed to collect resources", "type", job.Handler.GetName(), "namespace", job.Namespace, "duration", duration, "error", err)
			} else {
				log.Error("Failed to collect resources", "type", job.Handler.GetName(), "duration", duration, "error", err)
			}
			result.Error = err
			if job.Namespace != "" {
				log.Trace("Finished collection job", "type", job.Handler.GetName(), "namespace", job.Namespace, "duration", duration, "error", err)
			} else {
				log.Trace("Finished collection job", "type", job.Handler.GetName(), "duration", duration, "error", err)
			}
		} else {
			result.Resources = batch
			if len(batch) > 0 {
				if job.Namespace != "" {
					log.Debug("Collected resources", "type", job.Handler.GetName(), "namespace", job.Namespace, "count", len(batch), "duration", duration)
				} else {
					log.Debug("Collected resources", "type", job.Handler.GetName(), "count", len(batch), "duration", duration)
				}
			}
			if job.Namespace != "" {
				log.Trace("Finished collection job", "type", job.Handler.GetName(), "namespace", job.Namespace, "duration", duration, "count", len(batch))
			} else {
				log.Trace("Finished collection job", "type", job.Handler.GetName(), "duration", duration, "count", len(batch))
			}
		}

		select {
		case results <- result:
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			log.Error("Timeout sending results", "type", job.Handler.GetName(), "namespace", job.Namespace)
		}
	}
}

func generateCollectionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

func resourceKind(resource map[string]any) string {
	if resource == nil {
		return "unknown"
	}
	if kind, ok := resource["kind"].(string); ok && kind != "" {
		return strings.ToLower(kind)
	}
	return "unknown"
}
