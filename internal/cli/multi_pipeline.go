package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"bloodhound-kube/internal/multicluster"
	"bloodhound-kube/internal/utils"
)

// runSinglePipelineFn is the production default; tests replace it to avoid
// needing live clusters.
var runSinglePipelineFn = runSinglePipeline

type clusterResult struct {
	name       string
	jsonlPath  string
	parsedPath string
	nodeCount  int
	edgeCount  int
	duration   time.Duration
	err        error
}

func runMultiPipeline(ctx context.Context, req PipelineRequest, log utils.Logger) (PipelineResponse, error) {
	cfg, err := multicluster.LoadConfig(req.ClustersConfigPath)
	if err != nil {
		return PipelineResponse{}, err
	}
	if err := multicluster.ExpandEnvVars(cfg); err != nil {
		return PipelineResponse{}, err
	}
	if err := multicluster.Validate(cfg); err != nil {
		return PipelineResponse{}, err
	}
	entries := multicluster.ApplyDefaults(cfg)

	start := time.Now()
	results := make([]clusterResult, len(entries))

	for i, entry := range entries {
		clusterLog := log.With("cluster", entry.Name)
		clusterReq, err := buildClusterPipelineRequest(entry, req)
		if err != nil {
			results[i] = clusterResult{name: entry.Name, err: err}
			clusterLog.Error("Failed to build request", "error", err)
			continue
		}

		resp, err := safeRunSinglePipeline(ctx, clusterReq, clusterLog)
		results[i] = clusterResult{
			name:       entry.Name,
			jsonlPath:  resp.JSONLPath,
			parsedPath: resp.ParsedPath,
			nodeCount:  resp.NodeCount,
			edgeCount:  resp.EdgeCount,
			duration:   resp.Duration,
			err:        err,
		}
		if err != nil {
			clusterLog.Error("Cluster pipeline failed", "error", err)
		}
	}

	printMultiClusterSummary(results, req.ParseEnabled)

	var errs []error
	var totalNodes, totalEdges int
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.name, r.err))
		}
		totalNodes += r.nodeCount
		totalEdges += r.edgeCount
	}

	resp := PipelineResponse{
		NodeCount: totalNodes,
		EdgeCount: totalEdges,
		Duration:  time.Since(start),
	}
	if len(errs) > 0 {
		return resp, fmt.Errorf("%d of %d clusters failed: %w", len(errs), len(entries), errors.Join(errs...))
	}
	return resp, nil
}

func safeRunSinglePipeline(ctx context.Context, req PipelineRequest, log utils.Logger) (resp PipelineResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during collection: %v", r)
		}
	}()
	return runSinglePipelineFn(ctx, req, log)
}

func buildClusterPipelineRequest(entry multicluster.ClusterEntry, outer PipelineRequest) (PipelineRequest, error) {
	allNS := entry.AllNamespaces != nil && *entry.AllNamespaces
	redacted := entry.Redacted != nil && *entry.Redacted
	acceptCRDs := entry.AcceptCRDs != nil && *entry.AcceptCRDs

	jsonlPath, err := resolveClusterOutputPath(entry, outer)
	if err != nil {
		return PipelineRequest{}, err
	}
	parsedPath := ""
	if outer.ParseEnabled {
		parsedPath = resolveParsedOutputPath(jsonlPath, "")
	}

	return PipelineRequest{
		Collect: CollectRequest{
			Kubeconfig:         entry.Kubeconfig,
			Server:             entry.Server,
			Token:              entry.Token,
			ClusterType:        entry.ClusterType,
			Namespaces:         entry.Namespace,
			NamespaceFlagSet:   entry.Namespace != "",
			AllNamespaces:      allNS,
			Scope:              entry.Scope,
			DiscoveryAllowlist: entry.DiscoveryAllowlist,
			DiscoveryAccept:    acceptCRDs,
			Redacted:           redacted,
			Concurrency:        entry.Concurrency,
			PaginateLimit:      entry.PaginateLimit,
			Output:             jsonlPath,
			// global passthrough fields
			Resume:         outer.Collect.Resume,
			CheckpointFile: outer.Collect.CheckpointFile,
			FetchModeFull:  outer.Collect.FetchModeFull,
			ResourceTypes:  outer.Collect.ResourceTypes,
			DiscoveryList:  outer.Collect.DiscoveryList,
		},
		ParseEnabled:        outer.ParseEnabled,
		ParseUndefinedNodes: outer.ParseUndefinedNodes,
		ClusterName:         entry.Name,
		ParsedOutputPath:    parsedPath,
		ZipOutput:           outer.ZipOutput,
	}, nil
}

func resolveClusterOutputPath(entry multicluster.ClusterEntry, outer PipelineRequest) (string, error) {
	if entry.OutputFile != "" {
		return entry.OutputFile, nil
	}
	dir := entry.OutputDir
	if dir == "" {
		dir = outer.Collect.Output
		if dir == "" {
			dir = "."
		}
	}
	ts := time.Now().Format("2006-01-02-150405")
	filename := fmt.Sprintf("%s-%s.jsonl", entry.Name, ts)
	return filepath.Join(dir, filename), nil
}

func printMultiClusterSummary(results []clusterResult, parseEnabled bool) {
	succeeded := 0
	for _, r := range results {
		if r.err == nil {
			succeeded++
		}
	}
	total := len(results)
	fmt.Printf("\nMulti-cluster collection complete: %d/%d clusters succeeded\n", succeeded, total)

	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	for _, r := range results {
		if r.err != nil {
			if r.jsonlPath != "" {
				fmt.Fprintf(w, "  %s\tPARSE FAILED\t%s\t%v\n", r.name, filepath.Base(r.jsonlPath), r.err)
			} else {
				fmt.Fprintf(w, "  %s\tFAILED\t%v\n", r.name, r.err)
			}
			continue
		}
		jsonlBase := filepath.Base(r.jsonlPath)
		if parseEnabled && r.parsedPath != "" {
			fmt.Fprintf(w, "  %s\t%d nodes  %d edges\t%.1fs\t%s → %s\n",
				r.name, r.nodeCount, r.edgeCount, r.duration.Seconds(),
				jsonlBase, filepath.Base(r.parsedPath))
		} else {
			fmt.Fprintf(w, "  %s\t%d nodes  %d edges\t%.1fs\t%s\n",
				r.name, r.nodeCount, r.edgeCount, r.duration.Seconds(), jsonlBase)
		}
	}
	w.Flush()
}

// clusterOutputDir infers a directory from a path that may be a file or dir.
func clusterOutputDir(path string) string {
	if path == "" {
		return "."
	}
	if strings.HasSuffix(path, "/") || path == "." || path == ".." {
		return path
	}
	// If the path looks like a file (has an extension), return its directory.
	if filepath.Ext(path) != "" {
		return filepath.Dir(path)
	}
	return path
}
