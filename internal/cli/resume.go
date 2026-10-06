package cli

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"

	"bloodhound-kube/internal/collector"
)

func rejectResumeChanges(flags map[string]bool, incoming, saved map[string]any) error {
	for _, flag := range slices.Sorted(maps.Keys(incoming)) {
		value := incoming[flag]
		if flags[flag] && !reflect.DeepEqual(value, saved[flag]) {
			return fmt.Errorf("--%s conflicts with the saved run (saved value: %v); resume preserves the original plan and output settings", flag, saved[flag])
		}
	}
	return nil
}

func collectPlanFlags(req CollectRequest) map[string]any {
	return map[string]any{
		"namespace": req.Namespaces, "all-namespaces": req.AllNamespaces,
		"output": req.Output, "type": req.ResourceTypes, "cluster-type": req.ClusterType,
		"redacted": req.Redacted, "fetch-mode-full": req.FetchModeFull,
		"scope": req.Scope, "discovery-allowlist": req.DiscoveryAllowlist,
		"accept-crds": req.DiscoveryAccept, "discovery-list": req.DiscoveryList,
	}
}

func restoreCollectRequest(incoming CollectRequest, cp *collector.Checkpoint) (CollectRequest, error) {
	var saved CollectRequest
	if err := json.Unmarshal(cp.Settings, &saved); err != nil {
		return incoming, fmt.Errorf("invalid saved collection settings: %w", err)
	}
	if saved.Output != cp.OutputFile || saved.ClusterType == "" {
		return incoming, fmt.Errorf("checkpoint settings do not match the saved output or cluster")
	}
	if incoming.CheckpointFile == "" {
		return incoming, fmt.Errorf("--resume requires a checkpoint path")
	}
	if err := rejectResumeChanges(incoming.ExplicitFlags, collectPlanFlags(incoming), collectPlanFlags(saved)); err != nil {
		return incoming, err
	}
	if incoming.ExplicitFlags["checkpoint-file"] {
		return incoming, fmt.Errorf("--checkpoint-file is for new collections; use --resume <checkpoint> to resume")
	}
	if incoming.ExplicitFlags["concurrency"] {
		saved.Concurrency = incoming.Concurrency
	}
	if incoming.ExplicitFlags["paginate-limit"] {
		saved.PaginateLimit = incoming.PaginateLimit
	}
	if incoming.Kubeconfig != "" {
		saved.Kubeconfig = incoming.Kubeconfig
		saved.Server = ""
	}
	if incoming.Context != "" {
		saved.Context = incoming.Context
	}
	if incoming.Server != "" {
		saved.Server = incoming.Server
		saved.Kubeconfig, saved.Context = "", ""
	}
	saved.Token = incoming.Token
	if saved.Server != "" && saved.Token == "" {
		return incoming, fmt.Errorf("this run used token authentication; supply a fresh --token when resuming (tokens are not stored in checkpoints)")
	}
	saved.Resume, saved.CheckpointFile = true, incoming.CheckpointFile
	saved.ExplicitFlags = incoming.ExplicitFlags
	saved.PipelineSettings = incoming.PipelineSettings
	return saved, nil
}

func pipelineFlags(req PipelineRequest) map[string]any {
	return map[string]any{
		"no-parse": !req.ParseEnabled, "parsed-output": req.ParsedOutputPath,
		"cluster": req.ClusterName, "parse-undefined-nodes": req.ParseUndefinedNodes,
		"zip": req.ZipOutput,
	}
}

func restorePipelineRequest(incoming PipelineRequest) (PipelineRequest, error) {
	if incoming.ClustersConfigPath != "" {
		return incoming, fmt.Errorf("--resume identifies one cluster's checkpoint; omit --clusters-config when resuming")
	}
	cp, err := collector.LoadCheckpoint(incoming.Collect.CheckpointFile)
	if err != nil {
		return incoming, err
	}
	if err := cp.ValidateResume(); err != nil {
		return incoming, err
	}
	// Validate collection flags and credentials before starting any pipeline.
	if _, err := restoreCollectRequest(incoming.Collect, cp); err != nil {
		return incoming, err
	}
	if len(cp.Pipeline) == 0 {
		return incoming, fmt.Errorf("checkpoint has no saved pipeline settings; cannot restore output behavior")
	}
	var saved PipelineRequest
	if err := json.Unmarshal(cp.Pipeline, &saved); err != nil {
		return incoming, fmt.Errorf("invalid saved pipeline settings: %w", err)
	}
	if err := rejectResumeChanges(incoming.Collect.ExplicitFlags, pipelineFlags(incoming), pipelineFlags(saved)); err != nil {
		return incoming, err
	}
	saved.Collect, saved.Out = incoming.Collect, incoming.Out
	return saved, nil
}
