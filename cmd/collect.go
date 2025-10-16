package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"bloodhound-kube/internal/collector"
	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	namespace      string
	allNamespaces  bool
	logLevel       string
	output         string
	resourceTypes  []string
	concurrency    int
	timeout        int
	kubeconfig     string
	server         string
	token          string
	clusterType    string
	resume         bool
	checkpointFile string
)

// allResourceTypes will be populated after cluster detection
var allResourceTypes []string

func generateDefaultOutput() string {
	timestamp := time.Now().Format("2006-01-02-150405")
	return fmt.Sprintf("bloodhound-kube-%s.ndjson", timestamp)
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

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect Kubernetes resources",
	Long: `Collect Kubernetes resources from the cluster and stream as NDJSON

Authentication methods (in order of precedence):
1. --server and --token flags for direct API access
2. --kubeconfig flag to specify custom kubeconfig file  
3. KUBECONFIG environment variable
4. ~/.kube/config (default kubeconfig location)
5. In-cluster configuration (when running inside a pod)

Examples:
  # Use default kubeconfig
  bloodhound-kube collect

  # Use custom kubeconfig file
  bloodhound-kube collect --kubeconfig /path/to/config

  # Direct API access with token
  bloodhound-kube collect --server https://k8s-api.example.com --token eyJhbGciOi...

  # Specify cluster type (auto-detects by default)  
  bloodhound-kube collect --cluster-type openshift

  # Resume interrupted collection
  bloodhound-kube collect --resume

  # Resume with custom checkpoint file
  bloodhound-kube collect --resume --checkpoint-file /path/to/checkpoint.json

  # Specify custom output filename
  bloodhound-kube collect --output my-collection.ndjson

  # Specify output to a different directory
  bloodhound-kube collect --output /tmp/

  # Specify full path with directory and filename
  bloodhound-kube collect --output /tmp/my-collection.ndjson`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := utils.New(logLevel)

		if allNamespaces && cmd.Flags().Changed("namespace") {
			return fmt.Errorf("cannot use -A (all namespaces) and -n (namespace) flags together")
		}

		if (server != "" && token == "") || (server == "" && token != "") {
			return fmt.Errorf("--server and --token flags must be used together")
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
			return fmt.Errorf("invalid cluster type %q, must be one of: kubernetes, openshift, auto", clusterType)
		}

		cfg := utils.ClientConfig{
			Kubeconfig:  kubeconfig,
			Server:      server,
			Token:       token,
			ClusterType: clusterTypeEnum,
		}

		c, err := collector.New(cfg, log)
		if err != nil {
			return fmt.Errorf("failed to create collector: %w", err)
		}

		collector.DefaultRegistry.InitializeForCluster(c.GetClusterType())
		allResourceTypes = collector.DefaultRegistry.GetAllNames()

		log.Debug("Resource type selection", "inputResourceTypes", resourceTypes, "allResourceTypes", allResourceTypes)

		typesToCollect := resourceTypes
		if len(typesToCollect) == 0 {
			typesToCollect = allResourceTypes
			log.Debug("No specific types provided, using all available types", "typesToCollect", typesToCollect)
		} else {
			log.Debug("Using specific types provided", "typesToCollect", typesToCollect)
		}

		if err := collector.DefaultRegistry.ValidateTypes(typesToCollect); err != nil {
			return err
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
				return fmt.Errorf("checkpoint file not found: %s", defaultCheckpointPath)
			}

			existingCheckpoint, err = collector.LoadCheckpoint(defaultCheckpointPath)
			if err != nil {
				return fmt.Errorf("failed to load checkpoint: %w", err)
			}

			resumeFilename = existingCheckpoint.OutputFile
			checkpointFile = defaultCheckpointPath

			log.Info("Resuming collection", "checkpoint", checkpointFile, "output", resumeFilename)
			completed, total, pct := existingCheckpoint.GetProgress()
			log.Info("Previous progress", "completed", completed, "total", total, "percentage", fmt.Sprintf("%.1f%%", pct))
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()

		var namespacesToCollect []string
		if allNamespaces {
			namespacesToCollect, err = c.ListNamespaces(ctx)
			if err != nil {
				return fmt.Errorf("failed to list namespaces: %w", err)
			}
		} else {
			namespacesToCollect = []string{namespace}
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
				return fmt.Errorf("failed to create async writer for append: %w", err)
			}
			log.Info("Resuming collection, appending to existing file", "file", filename)
		} else {
			asyncWriter, err = utils.NewAsyncWriter(outputDir, filename, log)
			if err != nil {
				return fmt.Errorf("failed to create async writer: %w", err)
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
			return fmt.Errorf("collection completed with %d errors", len(errors))
		}

		return nil
	},
}

func init() {
	// Initialize registry to get available resource types
	registry := collector.NewResourceRegistry()
	// Initialize with Kubernetes defaults first
	registry.InitializeForCluster(utils.ClusterTypeKubernetes)
	availableTypes := registry.GetAllNames()

	// Create help text with dynamic resource types list
	resourceTypeHelp := fmt.Sprintf("Resource types to collect (%s, projects*, images*). *OpenShift only. Default: all types", strings.Join(availableTypes, ", "))

	collectCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	collectCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Collect from all namespaces (cannot be used with -n)")
	collectCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of concurrent workers for streaming collection")
	collectCmd.Flags().IntVarP(&timeout, "timeout", "", 300, "Timeout in seconds for the entire collection")
	collectCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
	collectCmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (can be directory, filename, or full path). Defaults to bloodhound-kube-YYYY-MM-DD-HHMMSS.ndjson in current directory")
	collectCmd.Flags().StringSliceVarP(&resourceTypes, "type", "t", []string{}, resourceTypeHelp)
	collectCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (overrides KUBECONFIG and ~/.kube/config)")
	collectCmd.Flags().StringVarP(&server, "server", "s", "", "Kubernetes API server address (requires --token)")
	collectCmd.Flags().StringVar(&token, "token", "", "Bearer token for authentication (requires --server)")
	collectCmd.Flags().StringVarP(&clusterType, "cluster-type", "T", "auto", "Cluster type: kubernetes, openshift, or auto (auto-detect)")
	collectCmd.Flags().BoolVar(&resume, "resume", false, "Resume from previous interrupted collection")
	collectCmd.Flags().StringVar(&checkpointFile, "checkpoint-file", "", "Path to checkpoint file (auto-generated if not specified)")

	rootCmd.AddCommand(collectCmd)
}
