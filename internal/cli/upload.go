package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"bloodhound-kube/config"
	"bloodhound-kube/internal/upload"
	"bloodhound-kube/internal/utils"
)

type UploadRequest struct {
	SchemaFile           string
	QueriesFile         string
	UploadFile          string
	BaseURL             string
	TokenID             string
	TokenKey            string
	Insecure            bool
	TimeoutSeconds      int
	Reset               bool
	ResetDB             bool
	HasSchemaFlag        bool
	HasQueriesFlag      bool
	HasUploadFlag       bool
	UseEmbeddedConfigs  bool
	ClusterName         string
}

type UploadService struct{}

func (s UploadService) Run(req UploadRequest, log utils.Logger) error {
	if req.TokenID == "" || req.TokenKey == "" {
		return fmt.Errorf("token ID and token key are required")
	}
	if !req.HasSchemaFlag && !req.HasQueriesFlag && !req.Reset && !req.HasUploadFlag && !req.UseEmbeddedConfigs {
		return fmt.Errorf("provide --schema-file, --queries-file, --upload-file, --configs, or --reset")
	}

	if req.UseEmbeddedConfigs {
		req.HasSchemaFlag = true
		req.SchemaFile = ""
		req.HasQueriesFlag = true
		req.QueriesFile = ""
	}

	client, err := upload.NewClient(upload.Config{
		BaseURL:            req.BaseURL,
		TokenID:            req.TokenID,
		TokenKey:           req.TokenKey,
		InsecureSkipVerify: req.Insecure,
		Timeout:            time.Duration(req.TimeoutSeconds) * time.Second,
		Logger:             log,
	})
	if err != nil {
		return fmt.Errorf("failed to create upload client: %w", err)
	}

	ctx := context.Background()
	if req.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	if req.ResetDB {
		if err := client.ResetDatabase(ctx); err != nil {
			return fmt.Errorf("failed to reset database: %w", err)
		}
		fmt.Println("Database reset successfully.")
	}

	if req.Reset && !req.HasSchemaFlag && !req.HasQueriesFlag && !req.ResetDB && !req.HasUploadFlag {
		if err := client.ResetExtensions(ctx); err != nil {
			return fmt.Errorf("failed to reset extensions: %w", err)
		}
		if err := client.ResetQueries(ctx); err != nil {
			return fmt.Errorf("failed to reset custom queries: %w", err)
		}
		fmt.Println("Custom data reset successfully.")
		return nil
	}

	if req.HasSchemaFlag {
		modelFile, cleanup, err := loadAndMergeSchema(req.SchemaFile)
		if err != nil {
			return fmt.Errorf("failed to load schema: %w", err)
		}
		if cleanup != nil {
			defer cleanup()
		}
		if req.Reset {
			if err := client.ResetExtensions(ctx); err != nil {
				return fmt.Errorf("failed to reset extensions: %w", err)
			}
		}
		if err := client.UploadExtension(ctx, modelFile); err != nil {
			return fmt.Errorf("failed to upload extension schema: %w", err)
		}
		fmt.Println("Extension schema uploaded successfully.")
	}

	if req.HasQueriesFlag {
		queriesFile, cleanup, err := loadAndMergeQueries(req.QueriesFile)
		if err != nil {
			return fmt.Errorf("failed to load queries: %w", err)
		}
		if cleanup != nil {
			defer cleanup()
		}
		if req.Reset {
			if err := client.ResetQueries(ctx); err != nil {
				return fmt.Errorf("failed to reset custom queries: %w", err)
			}
		}
		queryCount, err := client.UploadQueriesFromFile(ctx, queriesFile, req.ClusterName)
		if err != nil {
			return fmt.Errorf("failed to upload custom queries: %w", err)
		}
		fmt.Printf("Imported %d saved queries.\n", queryCount)
	}

	if req.HasUploadFlag {
		if req.UploadFile == "" {
			return fmt.Errorf("upload file is required when --upload-file is set")
		}
		if err := client.UploadOutput(ctx, req.UploadFile); err != nil {
			return fmt.Errorf("failed to upload collections: %w", err)
		}
	}
	return nil
}

func loadAndMergeQueries(userFile string) (string, func(), error) {
	embeddedData, _ := config.GetEmbeddedQueries()
	if userFile == "" {
		if embeddedData == nil {
			return "", nil, fmt.Errorf("no embedded queries available (build with -tags embedded)")
		}
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
		return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }, nil
	}

	userData, err := os.ReadFile(userFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read queries file: %w", err)
	}
	if embeddedData == nil {
		return userFile, nil, nil
	}
	mergedData, err := config.MergeQueries(embeddedData, userData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to merge queries: %w", err)
	}
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
	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }, nil
}

func loadAndMergeSchema(userFile string) (string, func(), error) {
	embeddedData, _ := config.GetEmbeddedSchema()
	if userFile == "" {
		if embeddedData == nil {
			return "", nil, fmt.Errorf("no embedded schema available (build with -tags embedded)")
		}
		tmpFile, err := os.CreateTemp("", "schema-*.json")
		if err != nil {
			return "", nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		if _, err := tmpFile.Write(embeddedData); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return "", nil, fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()
		return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }, nil
	}

	userData, err := os.ReadFile(userFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	if embeddedData == nil {
		return userFile, nil, nil
	}
	mergedData, err := config.MergeSchema(embeddedData, userData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to merge schema: %w", err)
	}
	tmpFile, err := os.CreateTemp("", "schema-merged-*.json")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	if _, err := tmpFile.Write(mergedData); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()
	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }, nil
}
