package cli

import (
	"bufio"
	"context"
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
	Server             string
	Token              string
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
}

type CollectResponse struct {
	OutputPath string
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
	if err := validateCollectRequest(req); err != nil {
		return CollectResponse{}, err
	}

	clusterTypeEnum, err := resolveClusterType(req.ClusterType)
	if err != nil {
		return CollectResponse{}, err
	}

	c, err := collector.New(utils.ClientConfig{Kubeconfig: req.Kubeconfig, Server: req.Server, Token: req.Token, ClusterType: clusterTypeEnum}, log, req.Redacted, req.PaginateLimit)
	if err != nil {
		return CollectResponse{}, fmt.Errorf("failed to create collector: %w", err)
	}

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
	targets, err := plan.TargetsForTypes(req.ResourceTypes)
	if err != nil {
		return CollectResponse{}, err
	}

	outputResolution, err := resolveOutputAndCheckpoint(req)
	if err != nil {
		return CollectResponse{}, err
	}

	var namespacesToCollect []string
	if req.AllNamespaces {
		namespacesToCollect, err = c.ListNamespaces(ctx)
	} else {
		namespacesToCollect, err = utils.ParseNamespaces(req.Namespaces, req.Kubeconfig)
	}
	if err != nil {
		return CollectResponse{}, err
	}

	outputDir, filename := outputResolution.outputDir, outputResolution.filename
	checkpointPath := outputResolution.checkpointPath
	existingCheckpoint := outputResolution.checkpoint

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

	duration, counts, totalCollected, errors := collector.RunCollectionWithCheckpoint(ctx, c, asyncWriter, targets, namespacesToCollect, filename, req.Concurrency, log, existingCheckpoint, checkpointPath)

	scopeMsg := fmt.Sprintf("from namespace %s", namespacesToCollect[0])
	if len(namespacesToCollect) > 1 {
		scopeMsg = fmt.Sprintf("from all namespaces (%d namespaces)", len(namespacesToCollect))
	}
	fmt.Fprintf(out, "Collected %d resources %s from %s cluster in %v and wrote to %s\n", totalCollected, scopeMsg, c.GetPlatform(), duration, filename)
	fmt.Fprintf(out, "Performance: %.1f resources/sec with %d workers\n", float64(totalCollected)/duration.Seconds(), req.Concurrency)
	for _, resourceType := range slices.Sorted(maps.Keys(counts)) {
		fmt.Fprintf(out, "  - %s: %d\n", resourceType, counts[resourceType])
	}
	outputPath := filepath.Join(outputDir, filename)
	if len(errors) > 0 {
		return CollectResponse{OutputPath: outputPath}, &PartialCollectionError{Count: len(errors)}
	}

	return CollectResponse{OutputPath: outputPath}, nil
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
			outputDir, outputFilename := parseOutputPath(req.Output)
			resolved.checkpointPath = collector.DefaultCheckpointPath(outputDir, outputFilename)
		}
		if _, err := os.Stat(resolved.checkpointPath); err != nil {
			return resolved, fmt.Errorf("checkpoint file not found: %s", resolved.checkpointPath)
		}
		checkpoint, err := collector.LoadCheckpoint(resolved.checkpointPath)
		if err != nil {
			return resolved, fmt.Errorf("failed to load checkpoint: %w", err)
		}
		resolved.checkpoint = checkpoint
		resolved.resumeFilename = checkpoint.OutputFile
	}

	if req.Resume && resolved.resumeFilename != "" {
		resolved.outputDir, _ = parseOutputPath(req.Output)
		resolved.filename = resolved.resumeFilename
	} else {
		resolved.outputDir, resolved.filename = parseOutputPath(req.Output)
	}
	if resolved.checkpointPath == "" {
		resolved.checkpointPath = collector.DefaultCheckpointPath(resolved.outputDir, resolved.filename)
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
