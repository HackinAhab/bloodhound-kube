package collector

import (
	"bloodhound-kube/internal/utils"
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
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

func RunCollection(ctx context.Context, c *Collector, w *utils.AsyncWriter, typesToCollect, namespacesToCollect []string, filename string, concurrency int, log logrus.FieldLogger) (time.Duration, map[string]int, int, []error) {
	startTime := time.Now()
	stats := NewCollectionStats()

	// Validate concurrency value, specify reasonable default if invalid
	if concurrency < 1 {
		concurrency = 4
		log.WithField("concurrency", concurrency).Warn("Invalid concurrency value, using default")
	}

	jobBufferSize := max(len(typesToCollect)*len(namespacesToCollect), 100)

	jobs := make(chan CollectionJob, jobBufferSize)
	results := make(chan []Resource, 200)

	var wg sync.WaitGroup

	// Spawn workers correctly
	for i := 0; i < concurrency; i++ {
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
				if err := w.WriteJSONLBatch(batchBuffer); err != nil {
					log.WithError(err).WithField("size", len(batchBuffer)).Error("Failed to write batch")
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

	// Track handlers already added to avoid duplicates from nicknames
	seenHandlers := make(map[ResourceHandler]bool)

	for _, resourceType := range typesToCollect {
		handler, err := DefaultRegistry.GetHandler(resourceType)
		if err != nil {
			log.WithError(err).WithField("type", resourceType).Error("Unknown resource type")
			continue
		}

		// Skip if already added this handler (nickname deduplication)
		if seenHandlers[handler] {
			log.WithFields(logrus.Fields{"type": resourceType, "handler": handler.GetName()}).Debug("Skipping duplicate handler")
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
		log.WithError(err).Error("Failed final flush")
	}

	counts, totalCollected, errors := stats.GetStats()
	duration := time.Since(startTime)

	if len(errors) > 0 {
		log.WithField("error_count", len(errors)).Error("Collection completed with errors")
		for i, err := range errors {
			if i < 5 {
				log.WithError(err).Error("Collection error")
			}
		}
		if len(errors) > 5 {
			log.WithField("count", len(errors)-5).Error("Additional errors suppressed")
		}
	}

	return duration, counts, totalCollected, errors
}

func RunCollectionWithCheckpoint(ctx context.Context, c *Collector, w *utils.AsyncWriter, typesToCollect, namespacesToCollect []string, filename string, concurrency int, log logrus.FieldLogger, existingCheckpoint *Checkpoint, checkpointFile string) (time.Duration, map[string]int, int, []error) {
	startTime := time.Now()
	stats := NewCollectionStats()

	clusterJobs := make([]CollectionJob, 0)
	namespacedJobs := make([]CollectionJob, 0)

	// Track handlers we've already added to avoid duplicates from nicknames
	seenHandlers := make(map[ResourceHandler]bool)

	for _, resourceType := range typesToCollect {
		handler, err := DefaultRegistry.GetHandler(resourceType)
		if err != nil {
			log.WithError(err).WithField("type", resourceType).Error("Unknown resource type")
			continue
		}

		// Skip if we've already added this handler (nickname deduplication)
		if seenHandlers[handler] {
			log.WithFields(logrus.Fields{"type": resourceType, "handler": handler.GetName()}).Debug("Skipping duplicate handler")
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

	log.WithFields(logrus.Fields{"total_jobs": len(allJobs), "completed_jobs": len(checkpoint.CompletedJobs), "pending_jobs": len(pendingJobs)}).Info("Collection plan")

	if len(pendingJobs) == 0 {
		log.Info("All jobs already completed")
		counts, totalCollected, errors := stats.GetStats()
		return time.Since(startTime), counts, totalCollected, errors
	}

	checkpoint.JobsRemaining = len(pendingJobs)

	// Validate concurrency value
	if concurrency < 1 {
		concurrency = 4 // sensible default
		log.WithField("concurrency", concurrency).Warn("Invalid concurrency value, using default")
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
				if err := w.WriteJSONLBatch(batchBuffer); err != nil {
					log.WithError(err).WithField("size", len(batchBuffer)).Error("Failed to write batch")
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
					log.WithError(err).Error("Failed to save checkpoint")
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
		log.WithError(err).Error("Failed final flush")
	}

	if err := checkpoint.Save(checkpointFile); err != nil {
		log.WithError(err).Error("Failed to save final checkpoint")
	}

	counts, totalCollected, errors := stats.GetStats()
	duration := time.Since(startTime)

	if len(errors) > 0 {
		log.WithField("error_count", len(errors)).Error("Collection completed with errors")
		for i, err := range errors {
			if i < 5 {
				log.WithError(err).Error("Collection error")
			}
		}
		if len(errors) > 5 {
			log.WithField("count", len(errors)-5).Error("Additional errors suppressed")
		}
	}

	completed, total, pct := checkpoint.GetProgress()
	log.WithFields(logrus.Fields{"completed": completed, "total": total, "percentage": fmt.Sprintf("%.1f%%", pct)}).Info("Final progress")

	// Remove checkpoint file after successful collection completion
	if len(errors) == 0 && completed == total {
		if err := RemoveCheckpoint(checkpointFile); err != nil {
			log.WithError(err).WithField("checkpoint_file", checkpointFile).Warn("Failed to remove checkpoint file after successful collection")
		} else {
			log.WithField("checkpoint_file", checkpointFile).Debug("Removed checkpoint file after successful collection")
		}
	}

	return duration, counts, totalCollected, errors
}

type CollectionResult struct {
	JobType   string
	Namespace string
	Resources []Resource
	Duration  time.Duration
	Error     error
}

func checkpointWorker(ctx context.Context, c *Collector, jobs <-chan CollectionJob, results chan<- CollectionResult, log logrus.FieldLogger) {
	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		startTime := time.Now()
		batch, err := job.Handler.Collect(ctx, c, job.Namespace)
		duration := time.Since(startTime)

		result := CollectionResult{
			JobType:   job.Handler.GetName(),
			Namespace: job.Namespace,
			Duration:  duration,
		}

		if err != nil {
			if job.Namespace != "" {
				log.WithError(err).WithFields(logrus.Fields{"type": job.Handler.GetName(), "namespace": job.Namespace, "duration": duration}).Error("Failed to collect resources")
			} else {
				log.WithError(err).WithFields(logrus.Fields{"type": job.Handler.GetName(), "duration": duration}).Error("Failed to collect resources")
			}
			result.Error = err
		} else {
			result.Resources = batch
			if len(batch) > 0 {
				if job.Namespace != "" {
					log.WithFields(logrus.Fields{"type": job.Handler.GetName(), "namespace": job.Namespace, "count": len(batch), "duration": duration}).Debug("Collected resources")
				} else {
					log.WithFields(logrus.Fields{"type": job.Handler.GetName(), "count": len(batch), "duration": duration}).Debug("Collected resources")
				}
			}
		}

		select {
		case results <- result:
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			log.WithFields(logrus.Fields{"type": job.Handler.GetName(), "namespace": job.Namespace}).Error("Timeout sending results")
		}
	}
}

func generateCollectionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

func collectWorker(ctx context.Context, c *Collector, jobs <-chan CollectionJob, results chan<- []Resource, log logrus.FieldLogger) {
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
				log.WithError(err).WithFields(logrus.Fields{"type": job.Handler.GetName(), "namespace": job.Namespace, "duration": time.Since(startTime)}).Error("Failed to collect resources")
			} else {
				log.WithError(err).WithFields(logrus.Fields{"type": job.Handler.GetName(), "duration": time.Since(startTime)}).Error("Failed to collect resources")
			}
			continue
		}

		if len(batch) > 0 {
			if job.Namespace != "" {
				log.WithFields(logrus.Fields{"type": job.Handler.GetName(), "namespace": job.Namespace, "count": len(batch), "duration": time.Since(startTime)}).Debug("Collected resources")
			} else {
				log.WithFields(logrus.Fields{"type": job.Handler.GetName(), "count": len(batch), "duration": time.Since(startTime)}).Debug("Collected resources")
			}

			select {
			case results <- batch:
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				log.WithField("size", len(batch)).Error("Timeout sending results batch")
			}
		}
	}
}
