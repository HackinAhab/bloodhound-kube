package collector

import (
	"context"
	"sync"
	"time"

	"bloodhound-kube/internal/logger"
	"bloodhound-kube/internal/writer"
)

type StreamedResource struct {
	Type      string `json:"type"`
	Namespace string `json:"namespace,omitempty"`
	Resource  any    `json:"resource"`
	Timestamp string `json:"timestamp"`
}

type CollectionJob struct {
	ResourceType string
	Namespace    string
}

func RunCollection(ctx context.Context, c *Collector, w *writer.AsyncWriter, typesToCollect, namespacesToCollect []string, filename string, concurrency int, log *logger.Logger) (time.Duration, map[string]int, int, []error) {
	startTime := time.Now()
	stats := NewCollectionStats()

	jobBufferSize := len(typesToCollect) * len(namespacesToCollect)
	if jobBufferSize < 100 {
		jobBufferSize = 100
	}

	jobs := make(chan CollectionJob, jobBufferSize)
	results := make(chan []StreamedResource, 200)

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
					if res, ok := item.(StreamedResource); ok {
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
		if resourceType == "nodes" {
			clusterJobs = append(clusterJobs, CollectionJob{ResourceType: resourceType, Namespace: ""})
		} else {
			for _, ns := range namespacesToCollect {
				namespacedJobs = append(namespacedJobs, CollectionJob{ResourceType: resourceType, Namespace: ns})
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

func collectWorker(ctx context.Context, c *Collector, jobs <-chan CollectionJob, results chan<- []StreamedResource, log *logger.Logger) {
	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		startTime := time.Now()
		timestamp := startTime.Format(time.RFC3339)
		batch := make([]StreamedResource, 0, 50)

		switch job.ResourceType {
		case "nodes":
			nodes, err := c.CollectNodes(ctx)
			if err != nil {
				log.Error("Failed to collect nodes", "error", err, "duration", time.Since(startTime))
				continue
			}
			for _, node := range nodes {
				batch = append(batch, StreamedResource{
					Type:      "node",
					Resource:  node,
					Timestamp: timestamp,
				})
			}
			log.Debug("Collected nodes", "count", len(nodes), "duration", time.Since(startTime))

		case "secrets":
			secrets, err := c.CollectSecrets(ctx, job.Namespace)
			if err != nil {
				log.Error("Failed to collect secrets", "namespace", job.Namespace, "error", err, "duration", time.Since(startTime))
				continue
			}
			for _, secret := range secrets {
				batch = append(batch, StreamedResource{
					Type:      "secret",
					Namespace: job.Namespace,
					Resource:  secret,
					Timestamp: timestamp,
				})
			}
			if len(secrets) > 0 {
				log.Debug("Collected secrets", "namespace", job.Namespace, "count", len(secrets), "duration", time.Since(startTime))
			}

		case "services":
			services, err := c.CollectServices(ctx, job.Namespace)
			if err != nil {
				log.Error("Failed to collect services", "namespace", job.Namespace, "error", err, "duration", time.Since(startTime))
				continue
			}
			for _, service := range services {
				batch = append(batch, StreamedResource{
					Type:      "service",
					Namespace: job.Namespace,
					Resource:  service,
					Timestamp: timestamp,
				})
			}
			if len(services) > 0 {
				log.Debug("Collected services", "namespace", job.Namespace, "count", len(services), "duration", time.Since(startTime))
			}

		case "ingresses":
			ingresses, err := c.CollectIngresses(ctx, job.Namespace)
			if err != nil {
				log.Error("Failed to collect ingresses", "namespace", job.Namespace, "error", err, "duration", time.Since(startTime))
				continue
			}
			for _, ingress := range ingresses {
				batch = append(batch, StreamedResource{
					Type:      "ingress",
					Namespace: job.Namespace,
					Resource:  ingress,
					Timestamp: timestamp,
				})
			}
			if len(ingresses) > 0 {
				log.Debug("Collected ingresses", "namespace", job.Namespace, "count", len(ingresses), "duration", time.Since(startTime))
			}

		case "gateways":
			gateways, err := c.CollectGateways(ctx, job.Namespace)
			if err != nil {
				log.Error("Failed to collect gateways", "namespace", job.Namespace, "error", err, "duration", time.Since(startTime))
				continue
			}
			for _, gateway := range gateways {
				batch = append(batch, StreamedResource{
					Type:      "gateway",
					Namespace: job.Namespace,
					Resource:  gateway,
					Timestamp: timestamp,
				})
			}
			if len(gateways) > 0 {
				log.Debug("Collected gateways", "namespace", job.Namespace, "count", len(gateways), "duration", time.Since(startTime))
			}

		case "rbac":
			rbac, err := c.CollectRBAC(ctx, job.Namespace)
			if err != nil {
				log.Error("Failed to collect RBAC resources", "namespace", job.Namespace, "error", err, "duration", time.Since(startTime))
				continue
			}

			for _, role := range rbac.Roles {
				batch = append(batch, StreamedResource{
					Type:      "role",
					Namespace: job.Namespace,
					Resource:  role,
					Timestamp: timestamp,
				})
			}
			for _, rb := range rbac.RoleBindings {
				batch = append(batch, StreamedResource{
					Type:      "role_binding",
					Namespace: job.Namespace,
					Resource:  rb,
					Timestamp: timestamp,
				})
			}
			for _, cr := range rbac.ClusterRoles {
				batch = append(batch, StreamedResource{
					Type:      "cluster_role",
					Resource:  cr,
					Timestamp: timestamp,
				})
			}
			for _, crb := range rbac.ClusterRoleBindings {
				batch = append(batch, StreamedResource{
					Type:      "cluster_role_binding",
					Resource:  crb,
					Timestamp: timestamp,
				})
			}
			for _, sa := range rbac.ServiceAccounts {
				batch = append(batch, StreamedResource{
					Type:      "service_account",
					Namespace: job.Namespace,
					Resource:  sa,
					Timestamp: timestamp,
				})
			}

			rbacCount := len(rbac.Roles) + len(rbac.RoleBindings) + len(rbac.ClusterRoles) + len(rbac.ClusterRoleBindings) + len(rbac.ServiceAccounts)
			if rbacCount > 0 {
				log.Debug("Collected RBAC resources", "namespace", job.Namespace, "count", rbacCount, "duration", time.Since(startTime))
			}
		}

		if len(batch) > 0 {
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
