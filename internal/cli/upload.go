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
	SchemaFile                string
	QueriesFile               string
	UploadFile                string
	BaseURL                   string
	TokenID                   string
	TokenKey                  string
	Insecure                  bool
	TimeoutSeconds            int
	Reset                     bool
	ResetDB                   bool
	HasSchemaFlag             bool
	HasQueriesFlag            bool
	HasUploadFlag             bool
	UseEmbeddedConfigs        bool
	ClusterName               string
	EnableOpenGraphExtensions bool
}

type UploadService struct{}

func (s UploadService) Run(req UploadRequest, log *utils.Logger) error {
	if req.TokenID == "" || req.TokenKey == "" {
		return fmt.Errorf("token ID and token key are required")
	}
	if !req.HasSchemaFlag && !req.HasQueriesFlag && !req.Reset && !req.HasUploadFlag && !req.UseEmbeddedConfigs && !req.EnableOpenGraphExtensions {
		return fmt.Errorf("provide --schema-file, --queries-file, --upload-file, --configs, --enable-extension, or --reset")
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

	if req.EnableOpenGraphExtensions {
		toggled, err := client.EnableOpenGraphExtensions(ctx)
		if err != nil {
			return fmt.Errorf("failed to enable opengraph extension management: %w", err)
		}
		if toggled {
			fmt.Println("OpenGraph extension management enabled.")
		} else {
			fmt.Println("OpenGraph extension management already enabled.")
		}
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
		modelFile, cleanup, err := loadAndMergeConfig(req.SchemaFile, "schema", config.GetEmbeddedSchema, config.MergeSchema)
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
		queriesFile, cleanup, err := loadAndMergeConfig(req.QueriesFile, "queries", config.GetEmbeddedQueries, config.MergeQueries)
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

// loadAndMergeConfig loads an embedded config of the given kind ("schema" or "queries"),
// optionally merging it with a user-provided file, and returns a path to the result
// (writing a temp file when embedded data is involved) plus a cleanup func.
func loadAndMergeConfig(userFile string, kind string, getEmbedded func() ([]byte, error), merge func([]byte, []byte) ([]byte, error)) (string, func(), error) {
	embeddedData, _ := getEmbedded()
	if userFile == "" {
		if embeddedData == nil {
			return "", nil, fmt.Errorf("no embedded %s available (build with -tags embedded)", kind)
		}
		return writeTempFile(kind+"-*.json", embeddedData)
	}

	userData, err := os.ReadFile(userFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read %s file: %w", kind, err)
	}
	if embeddedData == nil {
		return userFile, nil, nil
	}
	mergedData, err := merge(embeddedData, userData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to merge %s: %w", kind, err)
	}
	return writeTempFile(kind+"-merged-*.json", mergedData)
}

func writeTempFile(pattern string, data []byte) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()
	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }, nil
}
