package cmd

import (
	"context"
	"fmt"
	"strings"

	"kube-bloodhound/internal/collector"
	"kube-bloodhound/internal/logger"
	"kube-bloodhound/internal/writer"

	"github.com/spf13/cobra"
)

var (
	namespace     string
	logLevel      string
	outputPath    string
	resourceTypes []string
)

var allResourceTypes = []string{"secrets", "services", "ingresses", "gateways", "rbac"}

type CollectionResult struct {
	Namespace string         `json:"namespace"`
	Resources map[string]any `json:"resources"`
	Counts    map[string]int `json:"counts"`
}

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect Kubernetes resources",
	Long:  "Collect Kubernetes resources from the cluster and output as JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.New(logLevel)

		// Use all resource types if none specified
		typesToCollect := resourceTypes
		if len(typesToCollect) == 0 {
			typesToCollect = allResourceTypes
		}

		// Validate resource types
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

		filename := writer.GenerateFilename(namespace + "-" + strings.Join(typesToCollect, "-"))
		asyncWriter, err := writer.NewAsyncWriter(outputPath, filename, log)
		if err != nil {
			return fmt.Errorf("failed to create async writer: %w", err)
		}
		defer asyncWriter.Close()

		c, err := collector.New(log)
		if err != nil {
			return fmt.Errorf("failed to create collector: %w", err)
		}

		ctx := context.Background()
		result := CollectionResult{
			Namespace: namespace,
			Resources: make(map[string]any),
			Counts:    make(map[string]int),
		}

		totalCollected := 0

		for _, resourceType := range typesToCollect {
			log.Info("Collecting resource type", "type", resourceType)

			switch resourceType {
			case "secrets":
				secrets, err := c.CollectSecrets(ctx, namespace)
				if err != nil {
					return fmt.Errorf("failed to collect secrets: %w", err)
				}
				result.Resources["secrets"] = secrets
				result.Counts["secrets"] = len(secrets)
				totalCollected += len(secrets)

			case "services":
				services, err := c.CollectServices(ctx, namespace)
				if err != nil {
					return fmt.Errorf("failed to collect services: %w", err)
				}
				result.Resources["services"] = services
				result.Counts["services"] = len(services)
				totalCollected += len(services)
				
			case "ingresses":
				ingresses, err := c.CollectIngresses(ctx, namespace)
				if err != nil {
					return fmt.Errorf("failed to collect ingresses: %w", err)
				}
				result.Resources["ingresses"] = ingresses
				result.Counts["ingresses"] = len(ingresses)
				totalCollected += len(ingresses)
				
			case "gateways":
				gateways, err := c.CollectGateways(ctx, namespace)
				if err != nil {
					return fmt.Errorf("failed to collect gateways: %w", err)
				}
				result.Resources["gateways"] = gateways
				result.Counts["gateways"] = len(gateways)
				totalCollected += len(gateways)
				
			case "rbac":
				rbac, err := c.CollectRBAC(ctx, namespace)
				if err != nil {
					return fmt.Errorf("failed to collect RBAC resources: %w", err)
				}
				result.Resources["rbac"] = rbac
				// For RBAC, count all sub-resources
				rbacCount := len(rbac.Roles) + len(rbac.RoleBindings) + len(rbac.ClusterRoles) + len(rbac.ClusterRoleBindings) + len(rbac.ServiceAccounts)
				result.Counts["rbac"] = rbacCount
				totalCollected += rbacCount
			}
		}

		if err := asyncWriter.WriteJSON(result); err != nil {
			return fmt.Errorf("failed to write JSON output: %w", err)
		}

		if err := asyncWriter.Flush(); err != nil {
			return fmt.Errorf("failed to flush output: %w", err)
		}

		fmt.Printf("Successfully collected %d resources (%s) from namespace %s and wrote to %s\n",
			totalCollected, strings.Join(typesToCollect, ", "), namespace, filename)

		// Print counts by type
		for resourceType, count := range result.Counts {
			fmt.Printf("  - %s: %d\n", resourceType, count)
		}

		return nil
	},
}

func init() {
	collectCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	collectCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
	collectCmd.Flags().StringVarP(&outputPath, "output", "o", ".", "Output directory path (default: current directory)")
	collectCmd.Flags().StringSliceVarP(&resourceTypes, "type", "t", []string{}, "Resource types to collect (secrets, services, ingresses, gateways, rbac). Default: all types")

	rootCmd.AddCommand(collectCmd)
}
