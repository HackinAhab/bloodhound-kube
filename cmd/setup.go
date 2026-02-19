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
	setupUploadFile  string
	setupBaseURL     string
	setupTokenID     string
	setupTokenKey    string
	setupInsecure    bool
	setupTimeout     int
	setupLogLevel    string
	setupReset       bool
	setupResetDB     bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Upload Kubernetes OpenGraph data, node models, and saved queries to a BloodHound server",
	Long: `Upload Kubernetes OpenGraph data,node models, and saved queries to a BloodHound server using the API. This command can be used to set up a BloodHound instance with Kubernetes-specific data models and queries.

This command can reset existing custom data before uploading new models or
queries, depending on the flags provided.

Examples:
  # Upload custom types using a local BloodHound server
  bloodhound-kube setup --model-file bh-setup/custom_types.json --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY

  # Specify a custom BloodHound server URL
  bloodhound-kube setup --model-file bh-setup/custom_types.json --url https://bh.example.com:8080 --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY`,
	RunE: func(cmd *cobra.Command, args []string) error {
		effectiveLogLevel := setupLogLevel
		if !cmd.Flags().Changed("log") && globalLogLevel != "" {
			effectiveLogLevel = globalLogLevel
		}
		log := utils.New(effectiveLogLevel, globalNoColor)
		utils.SetDefaultLogger(log)

		log.Debug("Starting setup command", "logLevel", effectiveLogLevel)

		if setupTokenID == "" || setupTokenKey == "" {
			return fmt.Errorf("token ID and token key are required")
		}

		hasModel := cmd.Flags().Changed("model-file")
		hasQueries := cmd.Flags().Changed("queries-file")
		hasUpload := cmd.Flags().Changed("upload-file")
		if !hasModel && !hasQueries && !setupReset && !hasUpload {
			return fmt.Errorf("provide --model-file, --queries-file, --upload-file, or --reset")
		}

		client, err := setup.NewClient(setup.Config{
			BaseURL:            setupBaseURL,
			TokenID:            setupTokenID,
			TokenKey:           setupTokenKey,
			InsecureSkipVerify: setupInsecure,
			Timeout:            time.Duration(setupTimeout) * time.Second,
			Logger:             log,
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

		if setupResetDB {
			log.Info("Resetting entire database")
			if err := client.ResetDatabase(ctx); err != nil {
				return fmt.Errorf("failed to reset database: %w", err)
			}
			fmt.Println("Database reset successfully.")
		}

		if setupReset && !hasModel && !hasQueries && !setupResetDB && !hasUpload {
			log.Info("Resetting custom nodes")
			if err := client.ResetCustomNodes(ctx); err != nil {
				return fmt.Errorf("failed to reset custom nodes: %w", err)
			}
			log.Info("Resetting custom queries")
			if err := client.ResetQueries(ctx); err != nil {
				return fmt.Errorf("failed to reset custom queries: %w", err)
			}
			fmt.Println("Custom data reset successfully.")
			return nil
		}

		if hasModel {
			if setupModelFile == "" {
				return fmt.Errorf("model file is required when --model-file is set")
			}
			if setupReset {
				log.Info("Resetting custom nodes")
				if err := client.ResetCustomNodes(ctx); err != nil {
					return fmt.Errorf("failed to reset custom nodes: %w", err)
				}
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
			if setupReset {
				log.Info("Resetting custom queries")
				if err := client.ResetQueries(ctx); err != nil {
					return fmt.Errorf("failed to reset custom queries: %w", err)
				}
			}
			log.Info("Uploading custom queries", "file", setupQueriesFile)
			queryCount, err := client.UploadQueriesFromFile(ctx, setupQueriesFile)
			if err != nil {
				return fmt.Errorf("failed to upload custom queries: %w", err)
			}
			fmt.Printf("Imported %d saved queries.\n", queryCount)
		}

		if hasUpload {
			if setupUploadFile == "" {
				return fmt.Errorf("upload file is required when --upload-file is set")
			}
			log.Info("Uploading collections from file", "file", setupUploadFile)
			if err := client.UploadOutput(ctx, setupUploadFile); err != nil {
				return fmt.Errorf("failed to upload collections: %w", err)
			}
			log.Info("Collections uploaded successfully.")
		}
		return nil
	},
}

func init() {
	setupCmd.Flags().StringVar(&setupModelFile, "model-file", "", "Path to the model JSON file")
	setupCmd.Flags().StringVar(&setupQueriesFile, "queries-file", "config/custom_queries.json", "Path to the saved queries JSON file")
	setupCmd.Flags().StringVar(&setupUploadFile, "upload-file", "", "Path to the parsed collections JSON file for upload")
	setupCmd.Flags().StringVar(&setupBaseURL, "url", setup.DefaultBaseURL, "Base URL of the BloodHound instance")
	setupCmd.Flags().StringVar(&setupTokenID, "token-id", "", "API token ID for authentication")
	setupCmd.Flags().StringVar(&setupTokenKey, "token-key", "", "API token key for authentication")
	setupCmd.Flags().BoolVar(&setupInsecure, "insecure", true, "Skip TLS certificate verification")
	setupCmd.Flags().IntVar(&setupTimeout, "timeout", 30, "Timeout in seconds for setup operations")
	setupCmd.Flags().StringVarP(&setupLogLevel, "log", "l", "info", "Log level (debug, info, warn, error)")
	setupCmd.Flags().BoolVar(&setupReset, "reset", false, "Reset existing custom data before uploading")
	setupCmd.Flags().BoolVar(&setupResetDB, "reset-db", false, "Reset the entire database before uploading")

	rootCmd.AddCommand(setupCmd)
}
