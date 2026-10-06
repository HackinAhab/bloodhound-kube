package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"bloodhound-kube/internal/collector"
	"bloodhound-kube/internal/utils"
)

func resumeMultiPipeline(ctx context.Context, req PipelineRequest, log *utils.Logger) (PipelineResponse, error) {
	run, err := loadMultiRun(req.Collect.CheckpointFile)
	if err != nil {
		return PipelineResponse{}, err
	}
	if req.Collect.ExplicitFlags["checkpoint-file"] {
		return PipelineResponse{}, fmt.Errorf("use --resume <parent-checkpoint>, not --checkpoint-file")
	}
	for _, c := range run.Clusters {
		if c.State == "complete" {
			continue
		}
		if err := rejectResumeChanges(req.Collect.ExplicitFlags, collectPlanFlags(req.Collect), collectPlanFlags(c.Request.Collect)); err != nil {
			return PipelineResponse{}, err
		}
		if err := rejectResumeChanges(req.Collect.ExplicitFlags, pipelineFlags(req), pipelineFlags(c.Request)); err != nil {
			return PipelineResponse{}, err
		}
	}
	if req.Collect.Token != "" || req.Collect.Server != "" || req.Collect.Kubeconfig != "" || req.Collect.Context != "" {
		return PipelineResponse{}, fmt.Errorf("parent resume uses per-cluster credential references; use --clusters-config to refresh credentials")
	}
	if req.Collect.ExplicitFlags["cluster-concurrency"] {
		if req.ClusterConcurrency < 1 {
			return PipelineResponse{}, fmt.Errorf("--cluster-concurrency must be positive when resuming")
		}
		run.Concurrency = req.ClusterConcurrency
	}
	if req.ClustersConfigPath != "" {
		run.CredentialsPath, err = absoluteReference(req.ClustersConfigPath)
		if err != nil {
			return PipelineResponse{}, err
		}
	}
	return executeMultiRun(ctx, req, run, req.Collect.CheckpointFile, log)
}

func prepareRunCluster(c runCluster, run *multiRunCheckpoint, incoming PipelineRequest) (PipelineRequest, error) {
	req := c.Request
	req.Collect.ExplicitFlags = map[string]bool{}
	for _, flag := range []string{"concurrency", "paginate-limit"} {
		req.Collect.ExplicitFlags[flag] = incoming.Collect.ExplicitFlags[flag]
	}
	needsCredentials := true
	if _, err := os.Stat(req.Collect.CheckpointFile); err == nil {
		cp, err := collector.LoadCheckpoint(req.Collect.CheckpointFile)
		if err != nil {
			return req, err
		}
		if err := cp.ValidateResume(); err != nil {
			return req, err
		}
		if cp.OutputFile != req.Collect.Output {
			return req, fmt.Errorf("child checkpoint output does not match parent run")
		}
		if req.Collect.ExpectedAPIServer != "" && req.Collect.ExpectedAPIServer != cp.APIServer {
			return req, fmt.Errorf("child checkpoint cluster does not match parent run")
		}
		req.Collect.Resume = true
		needsCredentials = cp.JobsRemaining > 0 && (cp.Phase == "" || cp.Phase == "collecting")
	} else if !os.IsNotExist(err) {
		return req, err
	} else {
		if c.Phase == "collected" || c.Phase == "parsed" || c.Phase == "complete" {
			return req, fmt.Errorf("missing child checkpoint for cluster %q in phase %s", c.Name, c.Phase)
		}
		// Setup can fail before any checkpoint is created. Existing output is not
		// evidence of completion and must never be overwritten on parent resume.
		for _, path := range clusterRunPaths(req) {
			if _, err := os.Stat(path); err == nil {
				return req, fmt.Errorf("missing child checkpoint with existing run artifact %q", path)
			} else if !os.IsNotExist(err) {
				return req, err
			}
		}
	}
	if needsCredentials {
		path := run.ConfigPath
		override := run.CredentialsPath != ""
		if override {
			path = run.CredentialsPath
		}
		credentials, err := resolveRunCredentials(c, path, override)
		if err != nil {
			return req, err
		}
		req.Collect.Kubeconfig, req.Collect.Context = credentials.Kubeconfig, credentials.Context
		req.Collect.Server, req.Collect.Token = credentials.Server, credentials.Token
	}
	return req, nil
}

type runEvent struct {
	index  int
	result clusterResult
}

func executeMultiRun(ctx context.Context, incoming PipelineRequest, run *multiRunCheckpoint, path string, log *utils.Logger) (PipelineResponse, error) {
	start := time.Now()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	out := incoming.Out
	if out == nil {
		out = os.Stdout
	}
	fmt.Fprintf(out, "Run checkpoint: %s\nResume with: bloodhound-kube collect --resume %q\n", path, path)
	for i := range run.Clusters {
		if incoming.Collect.ExplicitFlags["concurrency"] {
			run.Clusters[i].Request.Collect.Concurrency = incoming.Collect.Concurrency
		}
		if incoming.Collect.ExplicitFlags["paginate-limit"] {
			run.Clusters[i].Request.Collect.PaginateLimit = incoming.Collect.PaginateLimit
		}
	}
	if err := run.save(path); err != nil {
		return PipelineResponse{}, err
	}
	results := make([]clusterResult, len(run.Clusters))
	for i, c := range run.Clusters {
		results[i] = clusterResult{name: c.Name, PipelineResponse: c.Result, state: c.State}
		results[i].Phase = c.Phase
	}
	events := make(chan runEvent, len(run.Clusters))
	active, next := 0, 0
	var persistenceErr error
	for next < len(run.Clusters) || active > 0 {
		for ctx.Err() == nil && persistenceErr == nil && active < run.Concurrency && next < len(run.Clusters) {
			i := next
			next++
			if run.Clusters[i].State == "complete" {
				continue
			}
			run.Clusters[i].State = "active"
			if err := run.save(path); err != nil {
				persistenceErr = err
				cancel()
				break
			}
			c := run.Clusters[i] // Workers only read their own immutable snapshot.
			active++
			go func() {
				clusterLog := log.With("cluster", c.Name)
				req, err := prepareRunCluster(c, run, incoming)
				var buf bytes.Buffer
				req.Out = &buf
				var resp PipelineResponse
				if err == nil {
					resp, err = safeRunSinglePipeline(ctx, req, clusterLog)
				}
				if resp.Phase == "" {
					resp.Phase = "setup"
					if err == nil {
						resp.Phase = "complete"
					}
				}
				events <- runEvent{index: i, result: clusterResult{name: c.Name, PipelineResponse: resp, err: err, buffered: buf.Bytes()}}
			}()
		}
		if active == 0 {
			break
		}
		event := <-events
		active--
		results[event.index] = event.result
		c := &run.Clusters[event.index]
		c.Result, c.Phase = event.result.PipelineResponse, event.result.Phase
		c.LastError = ""
		c.State = "complete"
		if event.result.err != nil {
			c.State = "failed"
			c.LastError = event.result.err.Error()
		}
		results[event.index].state = c.State
		if persistenceErr == nil {
			if err := run.save(path); err != nil {
				persistenceErr = err
				cancel()
			}
		}
	}
	var errs []error
	if persistenceErr != nil {
		errs = append(errs, fmt.Errorf("persist run progress: %w", persistenceErr))
	}
	completed, nodes, edges := 0, 0, 0
	for i, c := range run.Clusters {
		if c.State == "complete" {
			completed++
			if persistenceErr == nil {
				if err := collector.RemoveCheckpoint(c.Request.Collect.CheckpointFile); err != nil {
					errs = append(errs, err)
				}
			}
		} else if results[i].err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Name, results[i].err))
		} else if ctx.Err() != nil {
			results[i].err = ctx.Err()
			errs = append(errs, fmt.Errorf("%s: %w", c.Name, ctx.Err()))
		}
		if len(results[i].buffered) > 0 {
			out.Write(results[i].buffered)
		}
		nodes += c.Result.NodeCount
		edges += c.Result.EdgeCount
	}
	printMultiClusterSummaryTo(out, results)
	resp := PipelineResponse{NodeCount: nodes, EdgeCount: edges, Duration: time.Since(start), CheckpointPath: path, Phase: "complete"}
	if completed != len(run.Clusters) || len(errs) > 0 {
		resp.Phase = "collecting"
		fmt.Fprintf(out, "Run unfinished. Resume with: bloodhound-kube collect --resume %q\n", path)
		return resp, fmt.Errorf("%d of %d clusters failed or unfinished: %w", len(run.Clusters)-completed, len(run.Clusters), errors.Join(errs...))
	}
	if err := collector.RemoveCheckpoint(path); err != nil {
		return resp, err
	}
	return resp, nil
}

func printMultiClusterSummaryTo(out io.Writer, results []clusterResult) {
	completed := 0
	for _, r := range results {
		if r.err == nil && r.Phase == "complete" {
			completed++
		}
	}
	fmt.Fprintf(out, "\nMulti-cluster collection complete: %d/%d clusters succeeded\n", completed, len(results))
	for _, r := range results {
		status := "COMPLETE"
		if r.state == "queued" {
			status = "QUEUED"
		} else if r.err != nil {
			switch r.Phase {
			case "collecting":
				status = "COLLECTION INCOMPLETE"
			case "collected":
				status = "PARSE FAILED"
			case "parsed":
				status = "ZIP FAILED"
			default:
				status = "SETUP FAILED"
			}
		} else if r.Phase == "setup" {
			status = "QUEUED"
		}
		fmt.Fprintf(out, "  %s\t%s\t%s", r.name, status, r.JSONLPath)
		if r.ParsedPath != "" {
			fmt.Fprintf(out, " → %s", r.ParsedPath)
		}
		if r.err != nil {
			fmt.Fprintf(out, "\t%v", r.err)
		}
		fmt.Fprintln(out)
	}
}
