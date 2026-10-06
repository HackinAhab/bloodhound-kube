package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"bloodhound-kube/internal/collector"
	"bloodhound-kube/internal/utils"
)

const defaultCRDPromptThreshold = 1

// PartialCollectionError is returned by CollectService.Run when collection
// completes but one or more resource types could not be collected (e.g. due to
// permission errors).
type PartialCollectionError struct {
	Count int
}

func (e *PartialCollectionError) Error() string {
	return fmt.Sprintf("collection completed with %d error(s); some resources may be missing", e.Count)
}

type CollectRequest struct {
	Namespaces         string
	AllNamespaces      bool
	Output             string
	ResourceTypes      []string
	Concurrency        int
	PaginateLimit      int
	Kubeconfig         string
	Context            string
	ExpectedAPIServer  string
	Server             string
	Token              string `json:"-"`
	ClusterType        string
	Resume             bool
	CheckpointFile     string
	Redacted           bool
	FetchModeFull      bool
	DiscoveryList      bool
	DiscoveryAccept    bool
	DiscoveryAllowlist string
	Scope              string
	NamespaceFlagSet   bool
	ExplicitFlags      map[string]bool `json:"-"`
	PipelineSettings   json.RawMessage `json:"-"`
	RetainCheckpoint   bool            `json:"-"`
}

type CollectResponse struct {
	OutputPath     string
	CheckpointPath string
}

type CollectService struct{}

type collectDiscoveryPolicy struct {
	explicitTypes       bool
	allowlistEntries    []collector.AllowlistEntry
	filteredResources   []collector.DiscoveryResource
	discoveryListOnly   bool
	discoveredResources []collector.DiscoveryResource
}

type collectScope string

const (
	collectScopeCore      collectScope = "core"
	collectScopeAll       collectScope = "all"
	collectScopeAllowlist collectScope = "allowlist"
)

type outputCheckpointResolution struct {
	outputDir      string
	filename       string
	checkpointPath string
	resumeFilename string
	checkpoint     *collector.Checkpoint
}

func (s CollectService) Run(ctx context.Context, req CollectRequest, out io.Writer, log *utils.Logger) (CollectResponse, error) {
	if out == nil {
		out = os.Stdout
	}
	outputResolution, err := resolveOutputAndCheckpoint(req)
	if err != nil {
		return CollectResponse{}, err
	}
	if req.Resume {
		req, err = restoreCollectRequest(req, outputResolution.checkpoint)
		if err != nil {
			return CollectResponse{}, err
		}
	}
	if err := validateCollectRequest(req); err != nil {
		return CollectResponse{}, err
	}

	clusterTypeEnum, err := resolveClusterType(req.ClusterType)
	if err != nil {
		return CollectResponse{}, err
	}

	c, err := collector.New(utils.ClientConfig{Kubeconfig: req.Kubeconfig, Context: req.Context, Server: req.Server, Token: req.Token, ClusterType: clusterTypeEnum}, log, req.Redacted, req.PaginateLimit)
	if err != nil {
		return CollectResponse{}, fmt.Errorf("failed to create collector: %w", err)
	}

	server, effectiveKubeconfig, effectiveContext := c.Connection()
	if req.ExpectedAPIServer != "" && req.ExpectedAPIServer != server {
		return CollectResponse{}, fmt.Errorf("saved cluster API server %q does not match connected server %q", req.ExpectedAPIServer, server)
	}
	var targets []collector.CollectionTarget
	var namespacesToCollect []string
	if req.Resume {
		if server != outputResolution.checkpoint.APIServer {
			return CollectResponse{}, fmt.Errorf("resume cluster mismatch: saved API server %q, connected to %q", outputResolution.checkpoint.APIServer, server)
		}
		targets = outputResolution.checkpoint.Targets
		namespacesToCollect = outputResolution.checkpoint.Namespaces
	} else {
		discoveryPolicy, err := resolveDiscoveryPolicy(ctx, req, c, log)
		if err != nil {
			return CollectResponse{}, err
		}
		if discoveryPolicy.discoveryListOnly {
			printDiscoveryTable(discoveryPolicy.discoveredResources)
			return CollectResponse{}, nil
		}

		collectionsCfg, err := collector.BuildCollectionsConfigFromDiscovery(discoveryPolicy.filteredResources)
		if err != nil {
			return CollectResponse{}, fmt.Errorf("failed to build collections from discovery: %w", err)
		}
		if req.FetchModeFull {
			overrideCollectionsFetchMode(collectionsCfg, collector.FetchModeFull)
		}
		plan := collector.NewCollectionPlan(collectionsCfg)
		targets, err = plan.TargetsForTypes(req.ResourceTypes)
		if err != nil {
			return CollectResponse{}, err
		}

		if req.AllNamespaces {
			namespacesToCollect, err = c.ListNamespaces(ctx)
		} else {
			if req.Namespaces == "" && effectiveKubeconfig != "" {
				req.Namespaces, err = utils.GetContextNamespace(effectiveKubeconfig, effectiveContext)
				if err != nil {
					return CollectResponse{}, err
				}
			}
			namespacesToCollect, err = utils.ParseNamespaces(req.Namespaces, effectiveKubeconfig)
		}
		if err != nil {
			return CollectResponse{}, err
		}
		seen := make(map[string]bool)
		unique := namespacesToCollect[:0]
		for _, ns := range namespacesToCollect {
			if !seen[ns] {
				unique = append(unique, ns)
				seen[ns] = true
			}
		}
		namespacesToCollect = unique
	}

	outputDir, filename := outputResolution.outputDir, outputResolution.filename
	checkpointPath := outputResolution.checkpointPath
	existingCheckpoint := outputResolution.checkpoint
	if existingCheckpoint == nil {
		outputPath, err := filepath.Abs(filepath.Join(outputDir, filename))
		if err != nil {
			return CollectResponse{}, err
		}
		req.Output = outputPath
		req.ClusterType = string(c.GetClusterType())
		// Save resolved namespaces as well as the original namespace intent.
		if !req.AllNamespaces {
			req.Namespaces = strings.Join(namespacesToCollect, ",")
		}
		existingCheckpoint = c.CreateCheckpoint(outputPath, targets, namespacesToCollect)
		existingCheckpoint.Pipeline = req.PipelineSettings
		if len(existingCheckpoint.Pipeline) == 0 {
			// Direct CollectService callers produce JSONL without a parse pipeline.
			existingCheckpoint.Pipeline, err = json.Marshal(PipelineRequest{ParseEnabled: false})
			if err != nil {
				return CollectResponse{}, err
			}
		}
	}
	req.Kubeconfig, req.Context = effectiveKubeconfig, effectiveContext
	existingCheckpoint.Settings, err = json.Marshal(req)
	if err != nil {
		return CollectResponse{}, err
	}

	existingCheckpoint.Retain = req.RetainCheckpoint
	var asyncWriter *utils.AsyncWriter
	if req.Resume {
		asyncWriter, err = utils.NewAsyncWriterAppend(outputDir, filename, log)
	} else {
		asyncWriter, err = utils.NewAsyncWriter(outputDir, filename, log)
	}
	if err != nil {
		return CollectResponse{}, fmt.Errorf("failed to create async writer: %w", err)
	}
	defer asyncWriter.Close()
	if req.Resume {
		if err := asyncWriter.Restore(existingCheckpoint.OutputOffset); err != nil {
			return CollectResponse{}, fmt.Errorf("cannot restore output: %w", err)
		}
	}
	fmt.Fprintf(out, "Checkpoint: %s\nResume with: bloodhound-kube collect --resume %q\n", checkpointPath, checkpointPath)

	duration, counts, totalCollected, collectionErrors := collector.RunCollectionWithCheckpoint(ctx, c, asyncWriter, targets, namespacesToCollect, filename, req.Concurrency, log, existingCheckpoint, checkpointPath)

	scopeMsg := "from cluster scope"
	if len(namespacesToCollect) > 0 {
		scopeMsg = fmt.Sprintf("from namespace %s", namespacesToCollect[0])
	}
	if len(namespacesToCollect) > 1 {
		scopeMsg = fmt.Sprintf("from all namespaces (%d namespaces)", len(namespacesToCollect))
	}
	fmt.Fprintf(out, "Collected %d resources %s from %s cluster in %v and wrote to %s\n", totalCollected, scopeMsg, c.GetPlatform(), duration, filename)
	fmt.Fprintf(out, "Performance: %.1f resources/sec with %d workers\n", float64(totalCollected)/duration.Seconds(), req.Concurrency)
	for _, resourceType := range slices.Sorted(maps.Keys(counts)) {
		fmt.Fprintf(out, "  - %s: %d\n", resourceType, counts[resourceType])
	}
	outputPath := filepath.Join(outputDir, filename)
	if len(collectionErrors) > 0 {
		fmt.Fprintf(out, "Collection unfinished. Resume with: bloodhound-kube collect --resume %q\n", checkpointPath)
		if ctx.Err() != nil {
			return CollectResponse{OutputPath: outputPath, CheckpointPath: checkpointPath}, ctx.Err()
		}
		var commitErr *collector.CheckpointCommitError
		if joined := errors.Join(collectionErrors...); errors.As(joined, &commitErr) {
			return CollectResponse{OutputPath: outputPath, CheckpointPath: checkpointPath}, joined
		}
		return CollectResponse{OutputPath: outputPath, CheckpointPath: checkpointPath}, &PartialCollectionError{Count: len(collectionErrors)}
	}

	return CollectResponse{OutputPath: outputPath, CheckpointPath: checkpointPath}, nil
}

func validateCollectRequest(req CollectRequest) error {
	if req.AllNamespaces && req.NamespaceFlagSet {
		return fmt.Errorf("cannot use -A (all namespaces) and -n (namespace) flags together")
	}
	if (req.Server != "" && req.Token == "") || (req.Server == "" && req.Token != "") {
		return fmt.Errorf("--server and --token flags must be used together")
	}
	return nil
}

func resolveClusterType(clusterType string) (utils.ClusterType, error) {
	switch clusterType {
	case "kubernetes", "k8s":
		return utils.ClusterTypeKubernetes, nil
	case "openshift", "ocp":
		return utils.ClusterTypeOpenShift, nil
	case "auto", "":
		return utils.ClusterTypeAuto, nil
	default:
		return "", fmt.Errorf("invalid cluster type %q, must be one of: kubernetes, openshift, auto", clusterType)
	}
}

func resolveDiscoveryPolicy(ctx context.Context, req CollectRequest, c *collector.Collector, log *utils.Logger) (collectDiscoveryPolicy, error) {
	policy := collectDiscoveryPolicy{}
	policy.explicitTypes = len(req.ResourceTypes) > 0
	scope, err := resolveCollectScope(req)
	if err != nil {
		return policy, err
	}

	resources, err := c.Discover(ctx)
	if err != nil {
		return policy, fmt.Errorf("failed to discover resources: %w", err)
	}
	policy.discoveredResources = resources
	policy.filteredResources = resources
	policy.discoveryListOnly = req.DiscoveryList

	if req.DiscoveryAllowlist != "" {
		entries, err := collector.ParseAllowlistFile(req.DiscoveryAllowlist)
		if err != nil {
			return policy, fmt.Errorf("failed to read allowlist file: %w", err)
		}
		policy.allowlistEntries = entries
		log.Info("Using discovery allowlist file", "path", req.DiscoveryAllowlist, "entries", len(entries))
	}

	if policy.explicitTypes {
		return policy, nil
	}

	filtered, err := applyDiscoveryFilterByScope(resources, scope, policy.allowlistEntries)
	if err != nil {
		return policy, err
	}
	policy.filteredResources = filtered

	includeCRDs, err := shouldIncludeCRDs(req, filtered, log)
	if err != nil {
		return policy, err
	}
	if includeCRDs {
		return policy, nil
	}

	policy.filteredResources = filterCRDResources(policy.filteredResources, false)
	return policy, nil
}

func resolveCollectScope(req CollectRequest) (collectScope, error) {
	scope := strings.TrimSpace(strings.ToLower(req.Scope))
	if scope == "" {
		scope = string(collectScopeCore)
	}

	if scope == string(collectScopeCore) && req.DiscoveryAllowlist != "" {
		scope = string(collectScopeAllowlist)
	}

	s := collectScope(scope)
	switch s {
	case collectScopeCore, collectScopeAll, collectScopeAllowlist:
		if s == collectScopeAllowlist && strings.TrimSpace(req.DiscoveryAllowlist) == "" {
			return "", fmt.Errorf("--scope allowlist requires --discovery-allowlist")
		}
		return s, nil
	default:
		return "", fmt.Errorf("invalid scope %q, must be one of: core, all, allowlist", req.Scope)
	}
}

func applyDiscoveryFilterByScope(resources []collector.DiscoveryResource, scope collectScope, allowlistEntries []collector.AllowlistEntry) ([]collector.DiscoveryResource, error) {
	filtered := resources
	switch scope {
	case collectScopeAll:
		filtered = resources
	case collectScopeAllowlist:
		defaults, err := collector.DefaultDiscoveryAllowlist()
		if err != nil {
			return nil, fmt.Errorf("failed to load default allowlist: %w", err)
		}
		defaults = collector.MergeAllowlists(defaults, allowlistEntries)
		filtered = collector.FilterDiscoveredResources(resources, defaults)
	case collectScopeCore:
		defaults, err := collector.DefaultDiscoveryAllowlist()
		if err != nil {
			return nil, fmt.Errorf("failed to load default allowlist: %w", err)
		}
		filtered = collector.FilterDiscoveredResources(resources, defaults)
	default:
		return nil, fmt.Errorf("unsupported scope %q", scope)
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no resources matched discovery filters (%s)", scope)
	}
	return filtered, nil
}

func shouldIncludeCRDs(req CollectRequest, resources []collector.DiscoveryResource, log *utils.Logger) (bool, error) {
	crdCount := countCRDResources(resources)
	if crdCount < defaultCRDPromptThreshold || req.DiscoveryAccept {
		return true, nil
	}
	if isInteractive() {
		accepted, promptErr := promptForCRDs(resources)
		if promptErr != nil {
			return false, promptErr
		}
		return accepted, nil
	}
	log.Warn("Skipping CRDs in non-interactive mode", "crd_count", crdCount)
	return false, nil
}

func resolveOutputAndCheckpoint(req CollectRequest) (outputCheckpointResolution, error) {
	resolved := outputCheckpointResolution{}
	resolved.checkpointPath = req.CheckpointFile
	if req.Resume {
		if resolved.checkpointPath == "" {
			return resolved, fmt.Errorf("--resume requires a checkpoint path")
		}
		if _, err := os.Stat(resolved.checkpointPath); err != nil {
			return resolved, fmt.Errorf("checkpoint file not found: %s", resolved.checkpointPath)
		}
		checkpoint, err := collector.LoadCheckpoint(resolved.checkpointPath)
		if err != nil {
			return resolved, fmt.Errorf("failed to load checkpoint: %w", err)
		}
		resolved.checkpoint = checkpoint
		if err := checkpoint.ValidateResume(); err != nil {
			return resolved, err
		}
		info, err := os.Stat(checkpoint.OutputFile)
		if err != nil {
			return resolved, fmt.Errorf("cannot resume without original output file %q: %w", checkpoint.OutputFile, err)
		}
		if !info.Mode().IsRegular() || info.Size() < checkpoint.OutputOffset {
			return resolved, fmt.Errorf("output file does not contain checkpoint's committed byte position %d", checkpoint.OutputOffset)
		}
		resolved.resumeFilename = checkpoint.OutputFile
	}

	if req.Resume && resolved.resumeFilename != "" {
		resolved.outputDir = filepath.Dir(resolved.resumeFilename)
		resolved.filename = filepath.Base(resolved.resumeFilename)
	} else {
		resolved.outputDir, resolved.filename = parseOutputPath(req.Output)
	}
	if resolved.checkpointPath == "" {
		resolved.checkpointPath = collector.DefaultCheckpointPath(resolved.outputDir, resolved.filename)
	}
	checkpointAbs, err := filepath.Abs(resolved.checkpointPath)
	if err != nil {
		return resolved, err
	}
	outputAbs, err := filepath.Abs(filepath.Join(resolved.outputDir, resolved.filename))
	if err != nil {
		return resolved, err
	}
	if checkpointAbs == outputAbs {
		return resolved, fmt.Errorf("checkpoint and output file must have different paths")
	}
	if !req.Resume && !req.DiscoveryList {
		if _, err := os.Stat(resolved.checkpointPath); err == nil {
			return resolved, fmt.Errorf("checkpoint already exists: %s; continue it with --resume %q", resolved.checkpointPath, resolved.checkpointPath)
		} else if !os.IsNotExist(err) {
			return resolved, err
		}
	}
	return resolved, nil
}

func generateDefaultOutput() string {
	return fmt.Sprintf("bloodhound-kube-%s.jsonl", time.Now().Format("2006-01-02-150405"))
}

func parseOutputPath(output string) (dir, filename string) {
	if output == "" {
		return ".", generateDefaultOutput()
	}
	if strings.HasSuffix(output, "/") || output == "." || output == ".." {
		return output, generateDefaultOutput()
	}
	dir = filepath.Dir(output)
	filename = filepath.Base(output)
	if filepath.Ext(filename) == "" {
		filename += ".jsonl"
	}
	if dir == "." && !strings.Contains(output, "/") {
		dir = "."
	}
	return dir, filename
}

func overrideCollectionsFetchMode(cfg *collector.CollectionsConfig, mode collector.FetchMode) {
	if cfg == nil {
		return
	}
	for i := range cfg.Collections {
		cfg.Collections[i].FetchMode = mode
	}
}

func printDiscoveryTable(resources []collector.DiscoveryResource) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "API\tGROUP\tVERSION\tRESOURCE\tKIND\tNAMESPACED\tCRD")
	crdCount := 0
	for _, res := range resources {
		group := res.Group
		api := ""
		if group == "" {
			group = "core"
			api = fmt.Sprintf("%s/%s", res.Version, res.Resource)
		} else {
			api = fmt.Sprintf("%s/%s/%s", group, res.Version, res.Resource)
		}
		if res.IsCRD {
			crdCount++
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%t\t%t\n", api, group, res.Version, res.Resource, res.Kind, res.Namespaced, res.IsCRD)
	}
	writer.Flush()
	fmt.Printf("\nTotal resources: %d (CRDs: %d)\n", len(resources), crdCount)
}

func countCRDResources(resources []collector.DiscoveryResource) int {
	count := 0
	for _, res := range resources {
		if res.IsCRD {
			count++
		}
	}
	return count
}

func filterCRDResources(resources []collector.DiscoveryResource, includeCRDs bool) []collector.DiscoveryResource {
	if includeCRDs {
		return resources
	}
	filtered := make([]collector.DiscoveryResource, 0, len(resources))
	for _, res := range resources {
		if !res.IsCRD {
			filtered = append(filtered, res)
		}
	}
	return filtered
}

func isInteractive() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func promptForCRDs(resources []collector.DiscoveryResource) (bool, error) {
	groupCounts := make(map[string]int)
	crdCount := 0
	for _, res := range resources {
		if !res.IsCRD {
			continue
		}
		group := res.Group
		if group == "" {
			group = "core"
		}
		groupCounts[group]++
		crdCount++
	}
	if crdCount == 0 {
		return true, nil
	}
	groups := make([]struct {
		group string
		count int
	}, 0, len(groupCounts))
	for group, count := range groupCounts {
		groups = append(groups, struct {
			group string
			count int
		}{group: group, count: count})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].count == groups[j].count {
			return groups[i].group < groups[j].group
		}
		return groups[i].count > groups[j].count
	})
	maxGroups := min(len(groups), 5)
	groupSummary := make([]string, 0, maxGroups)
	for i := range maxGroups {
		groupSummary = append(groupSummary, fmt.Sprintf("%s (%d)", groups[i].group, groups[i].count))
	}
	fmt.Printf("Discovered %d CRD-backed resources across %d groups.\n", crdCount, len(groupCounts))
	if len(groupSummary) > 0 {
		fmt.Printf("Top groups: %s\n", strings.Join(groupSummary, ", "))
	}
	fmt.Print("Proceed with CRDs? [y/N]: ")
	response, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}
