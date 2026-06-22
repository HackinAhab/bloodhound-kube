package cmd

import (
	"bloodhound-kube/internal/cli"
	"bloodhound-kube/internal/upload"
	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	uploadSchemaFile   string
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
	uploadCluster     string
	uploadConfigs     bool
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload Kubernetes OpenGraph data, node models, and saved queries to a BloodHound server",
	Long: `Upload Kubernetes OpenGraph data, schema, and saved queries to a BloodHound server using the API. This command can be used to set up a BloodHound instance with Kubernetes-specific schema and queries.

This command can reset existing custom data before uploading new schema or
queries, depending on the flags provided.

Examples:
  # Upload schema using a local BloodHound server
  bloodhound-kube upload --schema-file schema.json --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY

  # Upload embedded configs only (requires embedded build)
  bloodhound-kube upload --configs --token-id $BLOODHOUND_TOKEN_ID --token-key $BLOODHOUND_TOKEN_KEY

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

		hasSchemaFlag := cmd.Flags().Changed("schema-file")
		hasQueriesFlag := cmd.Flags().Changed("queries-file")
		hasUpload := cmd.Flags().Changed("upload-file")
		return cli.UploadService{}.Run(cli.UploadRequest{
			SchemaFile:          uploadSchemaFile,
			QueriesFile:        uploadQueriesFile,
			UploadFile:         uploadUploadFile,
			BaseURL:            uploadBaseURL,
			TokenID:            uploadTokenID,
			TokenKey:           uploadTokenKey,
			Insecure:           uploadInsecure,
			TimeoutSeconds:     uploadTimeout,
			Reset:              uploadReset,
			ResetDB:            uploadResetDB,
			HasSchemaFlag:       hasSchemaFlag,
			HasQueriesFlag:     hasQueriesFlag,
			HasUploadFlag:      hasUpload,
			UseEmbeddedConfigs: uploadConfigs,
			ClusterName:        uploadCluster,
		}, log)
	},
}

func init() {
	uploadCmd.Flags().StringVar(&uploadSchemaFile, "schema-file", "", "Path to the schema JSON file (use '' for embedded only)")
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
	uploadCmd.Flags().StringVar(&uploadCluster, "cluster", "default", "Cluster name to substitute in saved query filters")
	uploadCmd.Flags().BoolVar(&uploadConfigs, "configs", false, "Upload embedded queries and schema (embedded build only; shorthand for --queries-file='' --schema-file='')")

	rootCmd.AddCommand(uploadCmd)
}
