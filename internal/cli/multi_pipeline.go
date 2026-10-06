package cli

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"bloodhound-kube/internal/multicluster"
	"bloodhound-kube/internal/utils"
)

// Tests replace the runner to exercise scheduling without live clusters.
var runSinglePipelineFn = runSinglePipeline

type clusterResult struct {
	name string
	PipelineResponse
	err      error
	buffered []byte
	state    string
}

func runMultiPipeline(ctx context.Context, req PipelineRequest, log *utils.Logger) (PipelineResponse, error) {
	run, path, err := createMultiRun(req)
	if err != nil {
		return PipelineResponse{}, err
	}
	return executeMultiRun(ctx, req, run, path, log)
}

func safeRunSinglePipeline(ctx context.Context, req PipelineRequest, log *utils.Logger) (resp PipelineResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during collection: %v", r)
		}
	}()
	return runSinglePipelineFn(ctx, req, log)
}

func buildClusterPipelineRequest(entry multicluster.ClusterEntry, outer PipelineRequest, runTimestamp string) PipelineRequest {
	allNS := entry.AllNamespaces != nil && *entry.AllNamespaces
	redacted := entry.Redacted != nil && *entry.Redacted
	acceptCRDs := entry.AcceptCRDs != nil && *entry.AcceptCRDs
	jsonlPath := resolveClusterOutputPath(entry, outer, runTimestamp)
	parsedPath := ""
	if outer.ParseEnabled {
		parsedPath = resolveParsedOutputPath(jsonlPath, "")
	}
	checkpointPath := outer.Collect.CheckpointFile
	if checkpointPath != "" {
		name := url.QueryEscape(entry.Name)
		checkpointPath = strings.TrimSuffix(checkpointPath, filepath.Ext(checkpointPath)) + "." + name + filepath.Ext(checkpointPath)
	}
	return PipelineRequest{
		Collect: CollectRequest{
			Kubeconfig: entry.Kubeconfig, Context: entry.Context, Server: entry.Server, Token: entry.Token,
			ClusterType: entry.ClusterType, Namespaces: entry.Namespace, NamespaceFlagSet: entry.Namespace != "",
			AllNamespaces: allNS, Scope: entry.Scope, DiscoveryAllowlist: entry.DiscoveryAllowlist,
			DiscoveryAccept: acceptCRDs, Redacted: redacted, Concurrency: entry.Concurrency,
			PaginateLimit: entry.PaginateLimit, Output: jsonlPath,
			Resume: outer.Collect.Resume, CheckpointFile: checkpointPath,
			FetchModeFull: outer.Collect.FetchModeFull, ResourceTypes: outer.Collect.ResourceTypes,
			DiscoveryList: outer.Collect.DiscoveryList,
		},
		ParseEnabled: outer.ParseEnabled, ParseUndefinedNodes: outer.ParseUndefinedNodes,
		ClusterName: entry.Name, ParsedOutputPath: parsedPath, ZipOutput: outer.ZipOutput,
	}
}

func resolveClusterOutputPath(entry multicluster.ClusterEntry, outer PipelineRequest, runTimestamp string) string {
	if entry.OutputFile != "" {
		return entry.OutputFile
	}
	dir := entry.OutputDir
	if dir == "" {
		dir = outer.Collect.Output
		if dir == "" {
			dir = "."
		}
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%s.jsonl", entry.Name, runTimestamp))
}
