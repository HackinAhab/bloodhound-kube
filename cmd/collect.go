package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bloodhound-kube/internal/collector"
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
)

var allResourceTypes = []string{"secrets", "services", "ingresses", "gateways", "rbac", "nodes"}

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect Kubernetes resources",
	Long:  "Collect Kubernetes resources from the cluster and stream as NDJSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.New(logLevel)

		if allNamespaces && cmd.Flags().Changed("namespace") {
			return fmt.Errorf("cannot use -A (all namespaces) and -n (namespace) flags together")
		}

		typesToCollect := resourceTypes
		if len(typesToCollect) == 0 {
			typesToCollect = allResourceTypes
		}

		for _, rt := range typesToCollect {
			found := false
			for _, valid := range allResourceTypes {
				if rt == valid {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unsupported resource type: %s (supported: %s)", rt, strings.Join(allResourceTypes, ", "))
			}
		}

		c, err := collector.New(log)
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

	rootCmd.AddCommand(collectCmd)
}
