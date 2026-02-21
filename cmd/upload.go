package cmd

import (
	"context"
	"fmt"
	"time"

	"bloodhound-kube/internal/upload"
	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	uploadModelFile   string
	uploadQueriesFile string
	uploadUploadFile  string
	uploadBaseURL     string
	uploadTokenID     string
	uploadTokenKey    string
	uploadInsecure    bool
	uploadTimeout     int
	uploadLogLevel    string
	uploadReset       bool
	uploadResetDB     bool
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload Kubernetes OpenGraph data, node models, and saved queries to a BloodHound server",
	Long: `Upload Kubernetes OpenGraph data,node models, and saved queries to a BloodHound server using the API. This command can be used to set up a BloodHound instance with Kubernetes-specific data models and queries.

This command can reset existing custom data before uploading new models or
queries, depending on the flags provided.

Examples:
  # Upload custom types using a local BloodHound server
  bloodhound-kube upload --model-file bh-setup/custom_types.json --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY

  # Specify a custom BloodHound server URL
  bloodhound-kube upload --model-file bh-setup/custom_types.json --url https://bh.example.com:8080 --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY`,
	RunE: func(cmd *cobra.Command, args []string) error {
		effectiveLogLevel := uploadLogLevel
		if !cmd.Flags().Changed("log") && globalLogLevel != "" {
			effectiveLogLevel = globalLogLevel
		}
		log, closeFn, err := buildLogger(effectiveLogLevel, true)
		if err != nil {
			return err
		}
		defer closeFn()
		utils.SetDefaultLogger(log)

		log.Debug("Starting upload command", "logLevel", effectiveLogLevel)

		if uploadTokenID == "" || uploadTokenKey == "" {
			return fmt.Errorf("token ID and token key are required")
		}

		hasModel := cmd.Flags().Changed("model-file")
		hasQueries := cmd.Flags().Changed("queries-file")
		hasUpload := cmd.Flags().Changed("upload-file")
		if !hasModel && !hasQueries && !uploadReset && !hasUpload {
			return fmt.Errorf("provide --model-file, --queries-file, --upload-file, or --reset")
		}

		client, err := upload.NewClient(upload.Config{
			BaseURL:            uploadBaseURL,
			TokenID:            uploadTokenID,
			TokenKey:           uploadTokenKey,
			InsecureSkipVerify: uploadInsecure,
			Timeout:            time.Duration(uploadTimeout) * time.Second,
			Logger:             log,
		})
		if err != nil {
			return fmt.Errorf("failed to create upload client: %w", err)
		}

		ctx := context.Background()
		var cancel context.CancelFunc
		if uploadTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(uploadTimeout)*time.Second)
			defer cancel()
		}

		if uploadResetDB {
			log.Info("Resetting entire database")
			if err := client.ResetDatabase(ctx); err != nil {
				return fmt.Errorf("failed to reset database: %w", err)
			}
			fmt.Println("Database reset successfully.")
		}

		if uploadReset && !hasModel && !hasQueries && !uploadResetDB && !hasUpload {
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
			if uploadModelFile == "" {
				return fmt.Errorf("model file is required when --model-file is set")
			}
			if uploadReset {
				log.Info("Resetting custom nodes")
				if err := client.ResetCustomNodes(ctx); err != nil {
					return fmt.Errorf("failed to reset custom nodes: %w", err)
				}
			}

			log.Info("Uploading model", "file", uploadModelFile)
			if err := client.UploadModel(ctx, uploadModelFile); err != nil {
				return fmt.Errorf("failed to upload model: %w", err)
			}
			fmt.Println("Custom nodes uploaded successfully.")
		}

		if hasQueries {
			if uploadQueriesFile == "" {
				return fmt.Errorf("queries file is required when --queries-file is set")
			}
			if uploadReset {
				log.Info("Resetting custom queries")
				if err := client.ResetQueries(ctx); err != nil {
					return fmt.Errorf("failed to reset custom queries: %w", err)
				}
			}
			log.Info("Uploading custom queries", "file", uploadQueriesFile)
			queryCount, err := client.UploadQueriesFromFile(ctx, uploadQueriesFile)
			if err != nil {
				return fmt.Errorf("failed to upload custom queries: %w", err)
			}
			fmt.Printf("Imported %d saved queries.\n", queryCount)
		}

		if hasUpload {
			if uploadUploadFile == "" {
				return fmt.Errorf("upload file is required when --upload-file is set")
			}
			log.Info("Uploading collections from file", "file", uploadUploadFile)
			if err := client.UploadOutput(ctx, uploadUploadFile); err != nil {
				return fmt.Errorf("failed to upload collections: %w", err)
			}
			log.Info("Collections uploaded successfully.")
		}
		return nil
	},
}

func init() {
	uploadCmd.Flags().StringVar(&uploadModelFile, "model-file", "", "Path to the model JSON file")
	uploadCmd.Flags().StringVar(&uploadQueriesFile, "queries-file", "config/custom_queries.json", "Path to the saved queries JSON file")
	uploadCmd.Flags().StringVar(&uploadUploadFile, "upload-file", "", "Path to the parsed collections JSON file for upload")
	uploadCmd.Flags().StringVar(&uploadBaseURL, "url", upload.DefaultBaseURL, "Base URL of the BloodHound instance")
	uploadCmd.Flags().StringVar(&uploadTokenID, "token-id", "", "API token ID for authentication")
	uploadCmd.Flags().StringVar(&uploadTokenKey, "token-key", "", "API token key for authentication")
	uploadCmd.Flags().BoolVar(&uploadInsecure, "insecure", true, "Skip TLS certificate verification")
	uploadCmd.Flags().IntVar(&uploadTimeout, "timeout", 30, "Timeout in seconds for upload operations")
	uploadCmd.Flags().StringVarP(&uploadLogLevel, "log", "l", "info", "Log level (trace, debug, info, warn, error)")
	uploadCmd.Flags().BoolVar(&uploadReset, "reset", false, "Reset existing custom data before uploading")
	uploadCmd.Flags().BoolVar(&uploadResetDB, "reset-db", false, "Reset the entire database before uploading")

	rootCmd.AddCommand(uploadCmd)
}
