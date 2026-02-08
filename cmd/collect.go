package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"bloodhound-kube/internal/collector"
	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	namespaces         string
	allNamespaces      bool
	logLevel           string
	output             string
	resourceTypes      []string
	concurrency        int
	timeout            int
	kubeconfig         string
	server             string
	token              string
	clusterType        string
	resume             bool
	checkpointFile     string
	redacted           bool
	discoveryList      bool
	discoveryAuto      bool
	discoveryAccept    bool
	discoveryAllowlist string
)

// allResourceTypes will be populated after cluster detection
var allResourceTypes []string

const defaultCRDPromptThreshold = 25

func generateDefaultOutput() string {
	timestamp := time.Now().Format("2006-01-02-150405")
	return fmt.Sprintf("bloodhound-kube-%s.jsonl", timestamp)
}

func parseOutputPath(output string) (dir, filename string) {
	if output == "" {
		return ".", generateDefaultOutput()
	}

	// If output is just a directory (ends with / or is a known directory)
	if strings.HasSuffix(output, "/") || output == "." || output == ".." {
		return output, generateDefaultOutput()
	}

	dir = filepath.Dir(output)
	filename = filepath.Base(output)

	// If no directory specified, use current directory
	if dir == "." && !strings.Contains(output, "/") {
		dir = "."
	}

	return dir, filename
}

func runCollect(cmd *cobra.Command, args []string, log *utils.Logger) (string, error) {
	log.Debug("Starting collection command")

	if allNamespaces && cmd.Flags().Changed("namespace") {
		return "", fmt.Errorf("cannot use -A (all namespaces) and -n (namespace) flags together")
	}

	if (server != "" && token == "") || (server == "" && token != "") {
		return "", fmt.Errorf("--server and --token flags must be used together")
	}

	var clusterTypeEnum utils.ClusterType
	switch clusterType {
	case "kubernetes", "k8s":
		clusterTypeEnum = utils.ClusterTypeKubernetes
	case "openshift", "ocp":
		clusterTypeEnum = utils.ClusterTypeOpenShift
	case "auto", "":
		clusterTypeEnum = utils.ClusterTypeAuto
	default:
		return "", fmt.Errorf("invalid cluster type %q, must be one of: kubernetes, openshift, auto", clusterType)
	}

	cfg := utils.ClientConfig{
		Kubeconfig:  kubeconfig,
		Server:      server,
		Token:       token,
		ClusterType: clusterTypeEnum,
	}

	c, err := collector.New(cfg, log)
	if err != nil {
		return "", fmt.Errorf("failed to create collector: %w", err)
	}

	// Set redacted flag on collector
	c.SetRedacted(redacted)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Get dynamic client for CRD support
	dynamicClient, err := c.GetDynamicClient()
	if err != nil {
		return "", fmt.Errorf("failed to get dynamic client: %w", err)
	}

	explicitTypes := len(resourceTypes) > 0

	discoveryAutoEnabled := discoveryAuto
	if !cmd.Flags().Changed("discovery-auto") && !explicitTypes && discoveryAllowlist == "" {
		discoveryAutoEnabled = true
	}

	if discoveryList {
		resources, err := collector.DiscoverResources(ctx, c.GetClients(), log)
		if err != nil {
			return "", fmt.Errorf("failed to discover resources: %w", err)
		}
		printDiscoveryTable(resources)
		return "", nil
	}

	var typesToCollect []string
	var usingDiscovery bool
	var allowlistEntries []collector.AllowlistEntry
	allowlistProvided := discoveryAllowlist != ""

	if allowlistProvided {
		allowlistEntries, err = collector.ParseAllowlistFile(discoveryAllowlist)
		if err != nil {
			return "", fmt.Errorf("failed to read allowlist file: %w", err)
		}
		log.Info("Using discovery allowlist file", "path", discoveryAllowlist, "entries", len(allowlistEntries))
	}

	resources, err := collector.DiscoverResources(ctx, c.GetClients(), log)
	if err != nil {
		return "", fmt.Errorf("failed to discover resources: %w", err)
	}
	usingDiscovery = true

	if explicitTypes {
		collectionsCfg, err := collector.BuildCollectionsConfigFromDiscovery(resources)
		if err != nil {
			return "", fmt.Errorf("failed to build collections from discovery: %w", err)
		}
		if err := collector.DefaultRegistry.InitializeFromConfig(c.GetClients(), log, collectionsCfg, dynamicClient); err != nil {
			return "", fmt.Errorf("failed to initialize collection registry: %w", err)
		}
		log.Info("Successfully initialized collection registry", "handlers", len(collectionsCfg.Collections), "discovery", usingDiscovery)
	} else {
		filteredResources := resources
		source := "all"
		if allowlistProvided {
			defaults, err := collector.DefaultDiscoveryAllowlist()
			if err != nil {
				return "", fmt.Errorf("failed to load default allowlist: %w", err)
			}
			defaults = collector.MergeAllowlists(defaults, allowlistEntries)
			filteredResources = collector.FilterDiscoveredResources(resources, defaults)
			source = "default-allowlist+file"
		} else if !discoveryAutoEnabled {
			defaults, err := collector.DefaultDiscoveryAllowlist()
			if err != nil {
				return "", fmt.Errorf("failed to load default allowlist: %w", err)
			}
			filteredResources = collector.FilterDiscoveredResources(resources, defaults)
			source = "default-allowlist"
		}

		if len(filteredResources) == 0 {
			return "", fmt.Errorf("no resources matched discovery filters (%s)", source)
		}

		includeCRDs := true
		crdCount := countCRDResources(filteredResources)
		if crdCount >= defaultCRDPromptThreshold && !discoveryAccept {
			if isInteractive() {
				accepted, err := promptForCRDs(filteredResources)
				if err != nil {
					return "", err
				}
				includeCRDs = accepted
			} else {
				includeCRDs = false
				log.Warn("Skipping CRDs in non-interactive mode", "crd_count", crdCount)
			}
		}
		if !includeCRDs {
			if discoveryAutoEnabled {
				defaults, err := collector.DefaultDiscoveryAllowlist()
				if err != nil {
					return "", fmt.Errorf("failed to load default allowlist: %w", err)
				}
				if allowlistProvided {
					defaults = collector.MergeAllowlists(defaults, allowlistEntries)
					source = "default-allowlist+file"
				} else {
					source = "default-allowlist"
				}
				filteredResources = collector.FilterDiscoveredResources(resources, defaults)
				log.Info("CRDs skipped, falling back to default collections", "source", source)
			} else {
				filteredResources = filterCRDResources(filteredResources, false)
			}
		}

		collectionsCfg, err := collector.BuildCollectionsConfigFromDiscovery(filteredResources)
		if err != nil {
			return "", fmt.Errorf("failed to build collections from discovery: %w", err)
		}
		if err := collector.DefaultRegistry.InitializeFromConfig(c.GetClients(), log, collectionsCfg, dynamicClient); err != nil {
			return "", fmt.Errorf("failed to initialize collection registry: %w", err)
		}
		log.Info("Successfully initialized collection registry", "handlers", len(collectionsCfg.Collections), "discovery", usingDiscovery, "source", source)
	}

	allResourceTypes = collector.DefaultRegistry.GetAllNames()

	log.Debug("Resource type selection", "inputResourceTypes", resourceTypes, "allResourceTypes", allResourceTypes)

	typesToCollect = resourceTypes
	if len(typesToCollect) == 0 {
		typesToCollect = allResourceTypes
		log.Debug("No specific types provided, using all available types", "typesToCollect", typesToCollect)
	} else {
		log.Debug("Using specific types provided", "typesToCollect", typesToCollect)
	}

	if err := collector.DefaultRegistry.ValidateTypes(typesToCollect); err != nil {
		return "", err
	}

	var existingCheckpoint *collector.Checkpoint
	var resumeFilename string

	if resume {
		var defaultCheckpointPath string
		if checkpointFile == "" {
			outputDir, outputFilename := parseOutputPath(output)
			defaultCheckpointPath = collector.DefaultCheckpointPath(outputDir, outputFilename)
		} else {
			defaultCheckpointPath = checkpointFile
		}

		if !collector.CheckpointExists(defaultCheckpointPath) {
			return "", fmt.Errorf("checkpoint file not found: %s", defaultCheckpointPath)
		}

		existingCheckpoint, err = collector.LoadCheckpoint(defaultCheckpointPath)
		if err != nil {
			return "", fmt.Errorf("failed to load checkpoint: %w", err)
		}

		resumeFilename = existingCheckpoint.OutputFile
		checkpointFile = defaultCheckpointPath

		log.Info("Resuming collection", "checkpoint", checkpointFile, "output", resumeFilename)
		completed, total, pct := existingCheckpoint.GetProgress()
		log.Info("Previous progress", "completed", completed, "total", total, "percentage", fmt.Sprintf("%.1f%%", pct))
	}

	var namespacesToCollect []string
	if allNamespaces {
		namespacesToCollect, err = c.ListNamespaces(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to list namespaces: %w", err)
		}
	} else {
		namespacesToCollect, err = utils.ParseNamespaces(namespaces, kubeconfig)
		if err != nil {
			return "", fmt.Errorf("failed to parse namespaces: %w", err)
		}
	}

	var outputDir, filename string
	if resume && resumeFilename != "" {
		// When resuming, use the filename from checkpoint but parse the output for directory
		outputDir, _ = parseOutputPath(output)
		filename = resumeFilename
	} else {
		outputDir, filename = parseOutputPath(output)
	}

	if checkpointFile == "" {
		checkpointFile = collector.DefaultCheckpointPath(outputDir, filename)
	}

	var asyncWriter *utils.AsyncWriter
	if resume {
		asyncWriter, err = utils.NewAsyncWriterAppend(outputDir, filename, log)
		if err != nil {
			return "", fmt.Errorf("failed to create async writer for append: %w", err)
		}
		log.Info("Resuming collection, appending to existing file", "file", filename)
	} else {
		asyncWriter, err = utils.NewAsyncWriter(outputDir, filename, log)
		if err != nil {
			return "", fmt.Errorf("failed to create async writer: %w", err)
		}
	}
	defer asyncWriter.Close()

	duration, counts, totalCollected, errors := collector.RunCollectionWithCheckpoint(ctx, c, asyncWriter, typesToCollect, namespacesToCollect, filename, concurrency, log, existingCheckpoint, checkpointFile)

	var scopeMsg string
	if len(namespacesToCollect) > 1 {
		scopeMsg = fmt.Sprintf("from all namespaces (%d namespaces)", len(namespacesToCollect))
	} else {
		scopeMsg = fmt.Sprintf("from namespace %s", namespacesToCollect[0])
	}

	fmt.Printf("Collected %d resources (%s) %s from %s cluster in %v and wrote to %s\n",
		totalCollected, strings.Join(typesToCollect, ", "), scopeMsg, c.GetPlatform(), duration, filename)

	resourcesPerSecond := float64(totalCollected) / duration.Seconds()
	fmt.Printf("Performance: %.1f resources/sec with %d workers\n", resourcesPerSecond, concurrency)

	for resourceType, count := range counts {
		fmt.Printf("  - %s: %d\n", resourceType, count)
	}

	if len(errors) > 0 {
		return "", fmt.Errorf("collection completed with %d errors", len(errors))
	}

	return filepath.Join(outputDir, filename), nil
}

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect Kubernetes resources",
	Long: `Collect Kubernetes resources from the cluster and stream as JSONL

Authentication methods (in order of precedence):
1. --server and --token flags for direct API access
2. --kubeconfig flag to specify custom kubeconfig file  
3. KUBECONFIG environment variable
4. ~/.kube/config (default kubeconfig location)
5. In-cluster configuration (when running inside a pod)

Examples:
  # Use default kubeconfig and current context namespace
  bloodhound-kube collect

  # Use custom kubeconfig file
  bloodhound-kube collect --kubeconfig /path/to/config
  # Specify single namespace
  bloodhound-kube collect --namespace production

  # Specify multiple namespaces
  bloodhound-kube collect --namespace prod,staging,dev

  # Direct API access with token
  bloodhound-kube collect --server https://k8s-api.example.com --token eyJhbGciOi...

  # Specify cluster type (auto-detects by default)  
  bloodhound-kube collect --cluster-type openshift

  # Resume interrupted collection
  bloodhound-kube collect --resume

  # Resume with custom checkpoint file
  bloodhound-kube collect --resume --checkpoint-file /path/to/checkpoint.json

  # Specify custom output filename
  bloodhound-kube collect --output my-collection.jsonl

  # Specify output to a different directory
  bloodhound-kube collect --output /tmp/

  # Specify full path with directory and filename
  bloodhound-kube collect --output /tmp/my-collection.jsonl`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Use local log level if set, otherwise use global log level
		effectiveLogLevel := logLevel
		if !cmd.Flags().Changed("log") && globalLogLevel != "" {
			effectiveLogLevel = globalLogLevel
		}
		log := utils.New(effectiveLogLevel)

		_, err := runCollect(cmd, args, log)
		return err
	},
}

func getAvailableResourcesHelp() string {
	return "Resource types to collect (defaults to discovered types)"
}

func init() {
	addCollectFlags(collectCmd)
	rootCmd.AddCommand(collectCmd)
}

func addCollectFlags(cmd *cobra.Command) {
	// Create help text with dynamic resource types list including nicknames
	resourceTypeHelp := getAvailableResourcesHelp()

	cmd.Flags().StringVarP(&namespaces, "namespace", "n", "", "Kubernetes namespace(s) - comma-delimited for multiple (defaults to current context namespace)")
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Collect from all namespaces (cannot be used with -n)")
	cmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of concurrent workers for streaming collection")
	cmd.Flags().IntVarP(&timeout, "timeout", "", 300, "Timeout in seconds for the entire collection")
	cmd.Flags().StringVarP(&logLevel, "log", "l", "info", "Log level (debug, info, warn, error)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (can be directory, filename, or full path). Defaults to bloodhound-kube-YYYY-MM-DD-HHMMSS.jsonl in current directory")
	cmd.Flags().StringSliceVarP(&resourceTypes, "type", "t", []string{}, resourceTypeHelp)
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (overrides KUBECONFIG and ~/.kube/config)")
	cmd.Flags().StringVarP(&server, "server", "s", "", "Kubernetes API server address (requires --token)")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token for authentication (requires --server)")
	cmd.Flags().StringVarP(&clusterType, "cluster-type", "T", "auto", "Cluster type: kubernetes, openshift, or auto (auto-detect)")
	cmd.Flags().BoolVar(&resume, "resume", false, "Resume from previous interrupted collection")
	cmd.Flags().StringVar(&checkpointFile, "checkpoint-file", "", "Path to checkpoint file (auto-generated if not specified)")
	cmd.Flags().BoolVar(&redacted, "redacted", false, "Redact secrets and sensitive data during collection")
	cmd.Flags().BoolVar(&discoveryList, "discovery-list", false, "List discovered API resources and exit")
	cmd.Flags().BoolVar(&discoveryAuto, "discovery-auto", false, "Collect all discovered resources when resources are not specified")
	cmd.Flags().BoolVar(&discoveryAccept, "discovery-auto-accept", false, "Automatically accept CRD discovery without prompting")
	cmd.Flags().StringVar(&discoveryAllowlist, "discovery-allowlist", "", "Path to newline-delimited allowlist of API resources (group/version/resource or group/resource)")
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
		if res.IsCRD {
			continue
		}
		filtered = append(filtered, res)
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

	var groupSummary []string
	for i := 0; i < maxGroups; i++ {
		groupSummary = append(groupSummary, fmt.Sprintf("%s (%d)", groups[i].group, groups[i].count))
	}

	fmt.Printf("Discovered %d CRD-backed resources across %d groups.\n", crdCount, len(groupCounts))
	if len(groupSummary) > 0 {
		fmt.Printf("Top groups: %s\n", strings.Join(groupSummary, ", "))
	}
	fmt.Print("Proceed with CRDs? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}
