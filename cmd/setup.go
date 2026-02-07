package cmd

import (
	"context"
	"fmt"
	"time"

	"bloodhound-kube/internal/setup"
	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	setupModelFile   string
	setupQueriesFile string
	setupBaseURL     string
	setupToken       string
	setupInsecure    bool
	setupTimeout     int
	setupLogLevel    string
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Upload custom BloodHound types and queries",
	Long: `Upload custom node models and saved queries to a BloodHound server.

This command clears existing custom nodes, uploads the provided model JSON, and
imports saved queries from the configured JSON file.

Examples:
  # Upload custom types using a local BloodHound server
  bloodhound-kube setup --model-file bh-setup/custom_types.json --token $BLOODHOUND_TOKEN

  # Use a remote BloodHound server with TLS verification disabled
  bloodhound-kube setup --model-file bh-setup/custom_types.json --url https://bh.example.com:8080 --token $BLOODHOUND_TOKEN --insecure`,
	RunE: func(cmd *cobra.Command, args []string) error {
		effectiveLogLevel := setupLogLevel
		if !cmd.Flags().Changed("log") && globalLogLevel != "" {
			effectiveLogLevel = globalLogLevel
		}
		log := utils.New(effectiveLogLevel)

		log.Debug("Starting setup command", "logLevel", effectiveLogLevel)

		if setupToken == "" {
			return fmt.Errorf("token is required")
		}

		hasModel := cmd.Flags().Changed("model-file")
		hasQueries := cmd.Flags().Changed("queries-file")
		if !hasModel && !hasQueries {
			return fmt.Errorf("provide --model-file, --queries-file, or both")
		}

		client, err := setup.NewClient(setup.Config{
			BaseURL:            setupBaseURL,
			Token:              setupToken,
			InsecureSkipVerify: setupInsecure,
			Timeout:            time.Duration(setupTimeout) * time.Second,
		})
		if err != nil {
			return fmt.Errorf("failed to create setup client: %w", err)
		}

		ctx := context.Background()
		var cancel context.CancelFunc
		if setupTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(setupTimeout)*time.Second)
			defer cancel()
		}

		if hasModel {
			if setupModelFile == "" {
				return fmt.Errorf("model file is required when --model-file is set")
			}
			log.Info("Resetting custom nodes")
			if err := client.ResetCustomNodes(ctx); err != nil {
				return fmt.Errorf("failed to reset custom nodes: %w", err)
			}

			log.Info("Uploading model", "file", setupModelFile)
			if err := client.UploadModel(ctx, setupModelFile); err != nil {
				return fmt.Errorf("failed to upload model: %w", err)
			}
			fmt.Println("Custom nodes uploaded successfully.")
		}

		if hasQueries {
			if setupQueriesFile == "" {
				return fmt.Errorf("queries file is required when --queries-file is set")
			}
			log.Info("Uploading custom queries", "file", setupQueriesFile)
			queryCount, err := client.UploadQueriesFromFile(ctx, setupQueriesFile)
			if err != nil {
				return fmt.Errorf("failed to upload custom queries: %w", err)
			}
			fmt.Printf("Imported %d saved queries.\n", queryCount)
		}
		return nil
	},
}

func init() {
	setupCmd.Flags().StringVar(&setupModelFile, "model-file", "", "Path to the model JSON file")
	setupCmd.Flags().StringVar(&setupQueriesFile, "queries-file", "config/custom_queries.json", "Path to the saved queries JSON file")
	setupCmd.Flags().StringVar(&setupBaseURL, "url", setup.DefaultBaseURL, "Base URL of the BloodHound instance")
	setupCmd.Flags().StringVar(&setupToken, "token", "", "API token for authentication")
	setupCmd.Flags().BoolVar(&setupInsecure, "insecure", true, "Skip TLS certificate verification")
	setupCmd.Flags().IntVar(&setupTimeout, "timeout", 30, "Timeout in seconds for setup operations")
	setupCmd.Flags().StringVarP(&setupLogLevel, "log", "l", "info", "Log level (debug, info, warn, error)")

	rootCmd.AddCommand(setupCmd)
}
