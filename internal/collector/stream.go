package collector

import (
	"context"
	"sync"
	"time"

	"bloodhound-kube/internal/logger"
	"bloodhound-kube/internal/writer"
)

type Resource struct {
	Type      string `json:"type"`
	Namespace string `json:"namespace,omitempty"`
	Resource  any    `json:"resource"`
	Timestamp string `json:"timestamp"`
}

type CollectionJob struct {
	Handler   ResourceHandler
	Namespace string
}

func RunCollection(ctx context.Context, c *Collector, w *writer.AsyncWriter, typesToCollect, namespacesToCollect []string, filename string, concurrency int, log *logger.Logger) (time.Duration, map[string]int, int, []error) {
	startTime := time.Now()
	stats := NewCollectionStats()

	jobBufferSize := len(typesToCollect) * len(namespacesToCollect)
	if jobBufferSize < 100 {
		jobBufferSize = 100
	}

	jobs := make(chan CollectionJob, jobBufferSize)
	results := make(chan []Resource, 200)

	var wg sync.WaitGroup

	for range concurrency {
		wg.Go(func() {
			collectWorker(ctx, c, jobs, results, log)
		})
	}

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		batchBuffer := make([]any, 0, 100)

		flushBatch := func() {
			if len(batchBuffer) > 0 {
				if err := w.WriteNDJSONBatch(batchBuffer); err != nil {
					log.Error("Failed to write batch", "error", err, "size", len(batchBuffer))
					stats.AddError(err)
				}
				for _, item := range batchBuffer {
					if res, ok := item.(Resource); ok {
						stats.AddCount(res.Type, 1)
					}
				}
				batchBuffer = batchBuffer[:0]
			}
		}

		flushTicker := time.NewTicker(50 * time.Millisecond)
		defer flushTicker.Stop()

		for {
			select {
			case resourceBatch, ok := <-results:
				if !ok {
					flushBatch()
					return
				}
				for _, resource := range resourceBatch {
					batchBuffer = append(batchBuffer, resource)
					if len(batchBuffer) >= 100 {
						flushBatch()
					}
				}
			case <-flushTicker.C:
				flushBatch()
			}
		}
	}()

	clusterJobs := make([]CollectionJob, 0)
	namespacedJobs := make([]CollectionJob, 0)

	for _, resourceType := range typesToCollect {
		handler, err := DefaultRegistry.GetHandler(resourceType)
		if err != nil {
			log.Error("Unknown resource type", "type", resourceType, "error", err)
			continue
		}

		if handler.IsClusterScoped() {
			clusterJobs = append(clusterJobs, CollectionJob{Handler: handler, Namespace: ""})
		} else {
			for _, ns := range namespacesToCollect {
				namespacedJobs = append(namespacedJobs, CollectionJob{Handler: handler, Namespace: ns})
			}
		}
	}

	for _, job := range clusterJobs {
		jobs <- job
	}

	for _, job := range namespacedJobs {
		jobs <- job
	}

	close(jobs)

	wg.Wait()
	close(results)

	<-writerDone

	if err := w.Flush(); err != nil {
		log.Error("Failed final flush", "error", err)
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

	return duration, counts, totalCollected, errors
}

func collectWorker(ctx context.Context, c *Collector, jobs <-chan CollectionJob, results chan<- []Resource, log *logger.Logger) {
	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		startTime := time.Now()
		batch, err := job.Handler.Collect(ctx, c, job.Namespace)
		if err != nil {
			if job.Namespace != "" {
				log.Error("Failed to collect resources", "type", job.Handler.GetName(), "namespace", job.Namespace, "error", err, "duration", time.Since(startTime))
			} else {
				log.Error("Failed to collect resources", "type", job.Handler.GetName(), "error", err, "duration", time.Since(startTime))
			}
			continue
		}

		if len(batch) > 0 {
			if job.Namespace != "" {
				log.Debug("Collected resources", "type", job.Handler.GetName(), "namespace", job.Namespace, "count", len(batch), "duration", time.Since(startTime))
			} else {
				log.Debug("Collected resources", "type", job.Handler.GetName(), "count", len(batch), "duration", time.Since(startTime))
			}

			select {
			case results <- batch:
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				log.Error("Timeout sending results batch", "size", len(batch))
			}
		}
	}
}
