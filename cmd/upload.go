package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"bloodhound-kube/config"
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
  bloodhound-kube upload --model-file custom_types.json --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY

  # Upload embedded configs only (requires embedded build)
  bloodhound-kube upload --queries-file='' --model-file='' --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY

  # Upload data only (no config changes)
  bloodhound-kube upload --upload-file data.json --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY`,
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

		hasModelFlag := cmd.Flags().Changed("model-file")
		hasQueriesFlag := cmd.Flags().Changed("queries-file")
		hasUpload := cmd.Flags().Changed("upload-file")
		
		if !hasModelFlag && !hasQueriesFlag && !uploadReset && !hasUpload {
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

		if uploadReset && !hasModelFlag && !hasQueriesFlag && !uploadResetDB && !hasUpload {
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

		if hasModelFlag {
			modelFile, cleanup, err := loadAndMergeModel(uploadModelFile, log)
			if err != nil {
				return fmt.Errorf("failed to load model: %w", err)
			}
			if cleanup != nil {
				defer cleanup()
			}
			
			if uploadReset {
				log.Info("Resetting custom nodes")
				if err := client.ResetCustomNodes(ctx); err != nil {
					return fmt.Errorf("failed to reset custom nodes: %w", err)
				}
			}

			log.Info("Uploading model", "file", modelFile)
			if err := client.UploadModel(ctx, modelFile); err != nil {
				return fmt.Errorf("failed to upload model: %w", err)
			}
			fmt.Println("Custom nodes uploaded successfully.")
		}

		if hasQueriesFlag {
			queriesFile, cleanup, err := loadAndMergeQueries(uploadQueriesFile, log)
			if err != nil {
				return fmt.Errorf("failed to load queries: %w", err)
			}
			if cleanup != nil {
				defer cleanup()
			}
			
			if uploadReset {
				log.Info("Resetting custom queries")
				if err := client.ResetQueries(ctx); err != nil {
					return fmt.Errorf("failed to reset custom queries: %w", err)
				}
			}
			
			log.Info("Uploading custom queries", "file", queriesFile)
			queryCount, err := client.UploadQueriesFromFile(ctx, queriesFile)
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

func loadAndMergeQueries(userFile string, log utils.Logger) (string, func(), error) {
	embeddedData, _ := config.GetEmbeddedQueries()
	
	if userFile == "" {
		// Empty string means use embedded only
		if embeddedData == nil {
			return "", nil, fmt.Errorf("no embedded queries available (build with -tags embedded)")
		}
		log.Info("Using embedded queries")
		
		// Write embedded data to temp file
		tmpFile, err := os.CreateTemp("", "queries-*.json")
		if err != nil {
			return "", nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		
		if _, err := tmpFile.Write(embeddedData); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return "", nil, fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()
		
		cleanup := func() { os.Remove(tmpFile.Name()) }
		return tmpFile.Name(), cleanup, nil
	}
	
	// User file path provided
	userData, err := os.ReadFile(userFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read queries file: %w", err)
	}
	
	if embeddedData == nil {
		// No embedded, use user only
		log.Info("Using user-provided queries", "file", userFile)
		return userFile, nil, nil
	}
	
	// Merge both (user takes precedence)
	log.Info("Merging user and embedded queries", "file", userFile)
	mergedData, err := config.MergeQueries(embeddedData, userData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to merge queries: %w", err)
	}
	
	// Write merged data to temp file
	tmpFile, err := os.CreateTemp("", "queries-merged-*.json")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	
	if _, err := tmpFile.Write(mergedData); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()
	
	cleanup := func() { os.Remove(tmpFile.Name()) }
	return tmpFile.Name(), cleanup, nil
}

func loadAndMergeModel(userFile string, log utils.Logger) (string, func(), error) {
	embeddedData, _ := config.GetEmbeddedTypes()
	
	if userFile == "" {
		// Empty string means use embedded only
		if embeddedData == nil {
			return "", nil, fmt.Errorf("no embedded model available (build with -tags embedded)")
		}
		log.Info("Using embedded model")
		
		// Write embedded data to temp file
		tmpFile, err := os.CreateTemp("", "model-*.json")
		if err != nil {
			return "", nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		
		if _, err := tmpFile.Write(embeddedData); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return "", nil, fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()
		
		cleanup := func() { os.Remove(tmpFile.Name()) }
		return tmpFile.Name(), cleanup, nil
	}
	
	// User file path provided
	userData, err := os.ReadFile(userFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read model file: %w", err)
	}
	
	if embeddedData == nil {
		// No embedded, use user only
		log.Info("Using user-provided model", "file", userFile)
		return userFile, nil, nil
	}
	
	// Merge both (user takes precedence)
	log.Info("Merging user and embedded model", "file", userFile)
	mergedData, err := config.MergeCustomTypes(embeddedData, userData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to merge model: %w", err)
	}
	
	// Write merged data to temp file
	tmpFile, err := os.CreateTemp("", "model-merged-*.json")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	
	if _, err := tmpFile.Write(mergedData); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()
	
	cleanup := func() { os.Remove(tmpFile.Name()) }
	return tmpFile.Name(), cleanup, nil
}

func init() {
	uploadCmd.Flags().StringVar(&uploadModelFile, "model-file", "", "Path to the model JSON file (use '' for embedded only)")
	uploadCmd.Flags().StringVar(&uploadQueriesFile, "queries-file", "", "Path to the saved queries JSON file (use '' for embedded only)")
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
