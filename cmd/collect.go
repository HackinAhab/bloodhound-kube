package cmd

import (
	"context"
	"os"

	"bloodhound-kube/internal/cli"
	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	namespaces          string
	allNamespaces       bool
	output              string
	resourceTypes       []string
	concurrency         int
	paginateLimit       int
	kubeconfig          string
	server              string
	token               string
	clusterType         string
	resume              bool
	checkpointFile      string
	redacted            bool
	fetchModeFull       bool
	discoveryList       bool
	discoveryAccept     bool
	discoveryAllowlist  string
	collectScope        string
	noParse             bool
	parsedOutput        string
	parseCluster        string
	parseUndefinedNodes bool
	clustersConfig      string
	zipOutput           bool
	clusterConcurrency  int
	parseInputFile      string
)

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect resources and output BloodHound JSON",
	Long: `Collect Kubernetes resources from the cluster, write to JSONL, and parse to BloodHound-compatible JSON.

Use --no-parse to write JSONL only.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log, closeFn, err := buildLogger(globalLogLevel, true)
		if err != nil {
			return err
		}
		defer closeFn()
		utils.SetDefaultLogger(log)

		if parseInputFile != "" {
			_, err = cli.ParseService{}.Run(cli.ParseRequest{
				InputPath:           parseInputFile,
				OutputPath:          parsedOutput,
				ClusterName:         parseCluster,
				ParseUndefinedNodes: parseUndefinedNodes,
			}, os.Stdout, log)
			return err
		}

		_, err = cli.PipelineService{}.Run(context.Background(), cli.PipelineRequest{
			Collect: cli.CollectRequest{
				Namespaces:         namespaces,
				AllNamespaces:      allNamespaces,
				Output:             output,
				ResourceTypes:      resourceTypes,
				Concurrency:        concurrency,
				PaginateLimit:      paginateLimit,
				Kubeconfig:         kubeconfig,
				Server:             server,
				Token:              token,
				ClusterType:        clusterType,
				Resume:             resume,
				CheckpointFile:     checkpointFile,
				Redacted:           redacted,
				FetchModeFull:      fetchModeFull,
				DiscoveryList:      discoveryList,
				DiscoveryAccept:    discoveryAccept,
				DiscoveryAllowlist: discoveryAllowlist,
				Scope:              collectScope,
				NamespaceFlagSet:   cmd.Flags().Changed("namespace"),
			},
			ParseEnabled:        !noParse,
			ParsedOutputPath:    parsedOutput,
			ClusterName:         parseCluster,
			ParseUndefinedNodes: parseUndefinedNodes,
			ClustersConfigPath:  clustersConfig,
			ZipOutput:           zipOutput,
			ClusterConcurrency:  clusterConcurrency,
		}, log)
		return err
	},
}

func init() {
	addCollectFlags(collectCmd)
	rootCmd.AddCommand(collectCmd)
}

func addCollectFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&namespaces, "namespace", "n", "", "Kubernetes namespace(s) - comma-delimited for multiple (defaults to current context namespace)")
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Collect from all namespaces (cannot be used with -n)")
	cmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of concurrent workers for streaming collection")
	cmd.Flags().IntVar(&paginateLimit, "paginate-limit", 100, "List pagination limit per API call")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (can be directory, filename, or full path). Defaults to bloodhound-kube-YYYY-MM-DD-HHMMSS.jsonl in current directory")
	cmd.Flags().StringSliceVarP(&resourceTypes, "type", "t", []string{}, "Resource types to collect (explicit override; accepts name, kind, shortnames, or API path)")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (overrides KUBECONFIG and ~/.kube/config)")
	cmd.Flags().StringVarP(&server, "server", "s", "", "Kubernetes API server address (requires --token)")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token for authentication (requires --server)")
	cmd.Flags().StringVarP(&clusterType, "cluster-type", "T", "auto", "Cluster type: kubernetes, openshift, or auto (auto-detect)")
	cmd.Flags().BoolVar(&resume, "resume", false, "Resume from previous interrupted collection")
	cmd.Flags().StringVar(&checkpointFile, "checkpoint-file", "", "Path to checkpoint file (auto-generated if not specified)")
	cmd.Flags().BoolVar(&redacted, "redacted", false, "Omit secret values during collection")
	cmd.Flags().BoolVar(&discoveryList, "discovery-list", false, "List discovered API resources and exit")
	cmd.Flags().BoolVar(&fetchModeFull, "fetch-mode-full", false, "Force full object fetch mode for all collected resources")
	cmd.Flags().StringVar(&collectScope, "scope", "core", "Collection scope: core, all, or allowlist (default: core)")
	cmd.Flags().BoolVar(&discoveryAccept, "accept-crds", false, "Automatically accept CRD discovery without prompting")
	cmd.Flags().StringVar(&discoveryAllowlist, "discovery-allowlist", "", "Path to newline-delimited allowlist of API resources (used by --scope allowlist)")
	cmd.Flags().BoolVar(&noParse, "no-parse", false, "Skip BloodHound JSON output (JSONL only)")
	cmd.Flags().StringVar(&parsedOutput, "parsed-output", "", "Output path for BloodHound JSON (defaults to JSONL filename with .json extension)")
	cmd.Flags().StringVar(&parseCluster, "cluster", "default", "Kubernetes cluster name for BloodHound metadata")
	cmd.Flags().BoolVar(&parseUndefinedNodes, "parse-undefined-nodes", false, "Enable generic node creation policy")
	cmd.Flags().StringVarP(&clustersConfig, "clusters-config", "C", "", "YAML config file for multi-cluster collection")
	cmd.Flags().BoolVar(&zipOutput, "zip", false, "Compress BloodHound JSON output into a zip archive")
	cmd.Flags().IntVar(&clusterConcurrency, "cluster-concurrency", 0, "Number of cluster pipelines to run in parallel (multi-cluster mode only; 0 defers to the clusters config clusterConcurrency default, or sequential if unset)")
	cmd.Flags().StringVar(&parseInputFile, "parse", "", "Parse an existing JSONL file to BloodHound JSON and exit (skips collection)")
}
