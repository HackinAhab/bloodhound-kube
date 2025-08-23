package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"kube-bloodhound/internal/collector"
	"kube-bloodhound/internal/logger"
)

var (
	namespace string
	logLevel  string
)

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect Kubernetes resources",
	Long:  "Collect Kubernetes resources from the cluster and output as JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.New(logLevel)
		
		c, err := collector.New(log)
		if err != nil {
			return fmt.Errorf("failed to create collector: %w", err)
		}

		ctx := context.Background()
		secrets, err := c.CollectSecrets(ctx, namespace)
		if err != nil {
			return fmt.Errorf("failed to collect secrets: %w", err)
		}

		output, err := json.MarshalIndent(secrets, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output: %w", err)
		}

		fmt.Println(string(output))
		return nil
	},
}

func init() {
	collectCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	collectCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
	
	rootCmd.AddCommand(collectCmd)
}