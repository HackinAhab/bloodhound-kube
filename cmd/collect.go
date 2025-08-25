package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bloodhound-kube/internal/collector"
	"bloodhound-kube/internal/k8s"
	"bloodhound-kube/internal/logger"
	"bloodhound-kube/internal/writer"

	"github.com/spf13/cobra"
)

var (
	namespace     string
	allNamespaces bool
	logLevel      string
	outputPath    string
	resourceTypes []string
	concurrency   int
	timeout       int
	kubeconfig    string
	server        string
	token         string
)

var allResourceTypes = collector.DefaultRegistry.GetAllNames()

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
  bloodhound-kube collect --server https://k8s-api.example.com --token eyJhbGciOi...`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.New(logLevel)

		if allNamespaces && cmd.Flags().Changed("namespace") {
			return fmt.Errorf("cannot use -A (all namespaces) and -n (namespace) flags together")
		}

		typesToCollect := resourceTypes
		if len(typesToCollect) == 0 {
			typesToCollect = allResourceTypes
		}

		if err := collector.DefaultRegistry.ValidateTypes(typesToCollect); err != nil {
			return err
		}

		// Validate authentication flags
		if (server != "" && token == "") || (server == "" && token != "") {
			return fmt.Errorf("--server and --token flags must be used together")
		}

		cfg := k8s.ClientConfig{
			Kubeconfig: kubeconfig,
			Server:     server,
			Token:      token,
		}

		c, err := collector.NewWithConfig(cfg, log)
		if err != nil {
			return fmt.Errorf("failed to create collector: %w", err)
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

		var filenamePrefix string
		if allNamespaces {
			filenamePrefix = "all-namespaces"
		} else {
			filenamePrefix = namespace
		}

		filename := writer.GenerateNDJSONFilename(filenamePrefix + "-" + strings.Join(typesToCollect, "-"))
		asyncWriter, err := writer.NewAsyncWriter(outputPath, filename, log)
		if err != nil {
			return fmt.Errorf("failed to create async writer: %w", err)
		}
		defer asyncWriter.Close()

		duration, counts, totalCollected, errors := collector.RunCollection(ctx, c, asyncWriter, typesToCollect, namespacesToCollect, filename, concurrency, log)

		var scopeMsg string
		if len(namespacesToCollect) > 1 {
			scopeMsg = fmt.Sprintf("from all namespaces (%d namespaces)", len(namespacesToCollect))
		} else {
			scopeMsg = fmt.Sprintf("from namespace %s", namespacesToCollect[0])
		}

		fmt.Printf("Collected %d resources (%s) %s in %v and wrote to %s\n",
			totalCollected, strings.Join(typesToCollect, ", "), scopeMsg, duration, filename)

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
	collectCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	collectCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Collect from all namespaces (cannot be used with -n)")
	collectCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of concurrent workers for streaming collection")
	collectCmd.Flags().IntVarP(&timeout, "timeout", "", 300, "Timeout in seconds for the entire collection")
	collectCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
	collectCmd.Flags().StringVarP(&outputPath, "output", "o", ".", "Output directory path")
	collectCmd.Flags().StringSliceVarP(&resourceTypes, "type", "t", []string{}, "Resource types to collect (secrets, services, ingresses, gateways, rbac, nodes). Default: all types")
	collectCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (overrides KUBECONFIG and ~/.kube/config)")
	collectCmd.Flags().StringVarP(&server, "server", "s", "", "Kubernetes API server address (requires --token)")
	collectCmd.Flags().StringVar(&token, "token", "", "Bearer token for authentication (requires --server)")

	rootCmd.AddCommand(collectCmd)
}
