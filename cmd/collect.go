package cmd

import (
	"context"
	"fmt"
	"strings"

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
)

var allResourceTypes = []string{"secrets", "services", "ingresses", "gateways", "rbac", "nodes"}

type CollectionResult struct {
	Namespace  string         `json:"namespace,omitempty"`
	Namespaces []string       `json:"namespaces,omitempty"`
	Resources  map[string]any `json:"resources"`
	Counts     map[string]int `json:"counts"`
}

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect Kubernetes resources",
	Long:  "Collect Kubernetes resources from the cluster and output as JSON",
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

		ctx := context.Background()

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
		filename := writer.GenerateFilename(filenamePrefix + "-" + strings.Join(typesToCollect, "-"))

		asyncWriter, err := writer.NewAsyncWriter(outputPath, filename, log)
		if err != nil {
			return fmt.Errorf("failed to create async writer: %w", err)
		}
		defer asyncWriter.Close()

		result := CollectionResult{
			Resources: make(map[string]any),
			Counts:    make(map[string]int),
		}

		if allNamespaces {
			result.Namespaces = namespacesToCollect
		} else {
			result.Namespace = namespace
		}

		totalCollected := 0

		for _, resourceType := range typesToCollect {
			log.Info("Collecting resource type", "type", resourceType)

			switch resourceType {
			case "nodes":
				nodes, err := c.CollectNodes(ctx)
				if err != nil {
					return fmt.Errorf("failed to collect nodes: %w", err)
				}
				result.Resources["nodes"] = nodes
				result.Counts["nodes"] = len(nodes)
				totalCollected += len(nodes)

			case "secrets", "services", "ingresses", "gateways", "rbac":
				var allResources []any
				totalCount := 0

				for _, ns := range namespacesToCollect {
					switch resourceType {
					case "secrets":
						secrets, err := c.CollectSecrets(ctx, ns)
						if err != nil {
							log.Error("Failed to collect secrets from namespace", "namespace", ns, "error", err)
							continue
						}
						for _, secret := range secrets {
							allResources = append(allResources, secret)
						}
						totalCount += len(secrets)

					case "services":
						services, err := c.CollectServices(ctx, ns)
						if err != nil {
							log.Error("Failed to collect services from namespace", "namespace", ns, "error", err)
							continue
						}
						for _, service := range services {
							allResources = append(allResources, service)
						}
						totalCount += len(services)

					case "ingresses":
						ingresses, err := c.CollectIngresses(ctx, ns)
						if err != nil {
							log.Error("Failed to collect ingresses from namespace", "namespace", ns, "error", err)
							continue
						}
						for _, ingress := range ingresses {
							allResources = append(allResources, ingress)
						}
						totalCount += len(ingresses)

					case "gateways":
						gateways, err := c.CollectGateways(ctx, ns)
						if err != nil {
							log.Error("Failed to collect gateways from namespace", "namespace", ns, "error", err)
							continue
						}
						for _, gateway := range gateways {
							allResources = append(allResources, gateway)
						}
						totalCount += len(gateways)

					case "rbac":
						rbac, err := c.CollectRBAC(ctx, ns)
						if err != nil {
							log.Error("Failed to collect RBAC resources from namespace", "namespace", ns, "error", err)
							continue
						}

						if result.Resources["rbac"] == nil {
							result.Resources["rbac"] = map[string]interface{}{
								"roles":                 []any{},
								"role_bindings":         []any{},
								"cluster_roles":         []any{},
								"cluster_role_bindings": []any{},
								"service_accounts":      []any{},
							}
						}
						rbacResult := result.Resources["rbac"].(map[string]any)

						for _, role := range rbac.Roles {
							rbacResult["roles"] = append(rbacResult["roles"].([]any), role)
						}
						for _, rb := range rbac.RoleBindings {
							rbacResult["role_bindings"] = append(rbacResult["role_bindings"].([]any), rb)
						}
						for _, cr := range rbac.ClusterRoles {
							rbacResult["cluster_roles"] = append(rbacResult["cluster_roles"].([]any), cr)
						}
						for _, crb := range rbac.ClusterRoleBindings {
							rbacResult["cluster_role_bindings"] = append(rbacResult["cluster_role_bindings"].([]any), crb)
						}
						for _, sa := range rbac.ServiceAccounts {
							rbacResult["service_accounts"] = append(rbacResult["service_accounts"].([]any), sa)
						}

						rbacCount := len(rbac.Roles) + len(rbac.RoleBindings) + len(rbac.ClusterRoles) + len(rbac.ClusterRoleBindings) + len(rbac.ServiceAccounts)
						totalCount += rbacCount
					}
				}

				if resourceType != "rbac" {
					result.Resources[resourceType] = allResources
				}
				result.Counts[resourceType] = totalCount
				totalCollected += totalCount
			}
		}

		if err := asyncWriter.WriteJSON(result); err != nil {
			return fmt.Errorf("failed to write JSON output: %w", err)
		}

		if err := asyncWriter.Flush(); err != nil {
			return fmt.Errorf("failed to flush output: %w", err)
		}

		var scopeMsg string
		if allNamespaces {
			scopeMsg = fmt.Sprintf("from all namespaces (%d namespaces)", len(namespacesToCollect))
		} else {
			scopeMsg = fmt.Sprintf("from namespace %s", namespace)
		}

		fmt.Printf("Collected %d resources (%s) %s and wrote to %s\n",
			totalCollected, strings.Join(typesToCollect, ", "), scopeMsg, filename)

		for resourceType, count := range result.Counts {
			fmt.Printf("  - %s: %d\n", resourceType, count)
		}

		return nil
	},
}

func init() {
	collectCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	collectCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Collect from all namespaces (cannot be used with -n)")
	collectCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
	collectCmd.Flags().StringVarP(&outputPath, "output", "o", ".", "Output directory path")
	collectCmd.Flags().StringSliceVarP(&resourceTypes, "type", "t", []string{}, "Resource types to collect (secrets, services, ingresses, gateways, rbac, nodes). Default: all types")

	rootCmd.AddCommand(collectCmd)
}
