package cmd

import (
	"path/filepath"
	"strings"

	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	parsedOutput string
)

func deriveParsedOutputPath(jsonlPath string) string {
	if strings.HasSuffix(jsonlPath, ".jsonl") {
		return strings.TrimSuffix(jsonlPath, ".jsonl") + ".json"
	}
	return jsonlPath + ".json"
}

func resolveParsedOutputPath(jsonlPath, parsedOutputPath string) string {
	if parsedOutputPath == "" {
		return deriveParsedOutputPath(jsonlPath)
	}

	if strings.HasSuffix(parsedOutputPath, "/") || parsedOutputPath == "." || parsedOutputPath == ".." {
		return filepath.Join(parsedOutputPath, filepath.Base(deriveParsedOutputPath(jsonlPath)))
	}

	return parsedOutputPath
}

var collectParseCmd = &cobra.Command{
	Use:   "collect-parse",
	Short: "Collect resources and parse to BloodHound format",
	Long: `Collect Kubernetes resources and parse them to BloodHound format.

This runs the collect phase first, writes JSONL to disk, then parses the
JSONL into BloodHound-compatible JSON output. The JSONL file is retained.

Examples:
  # Collect and parse with default output names
  bloodhound-kube collect-parse

  # Specify the JSONL output directory (parsed output uses same base name)
  bloodhound-kube collect-parse --output /tmp/

  # Specify the JSONL output filename (parsed output uses same base name)
  bloodhound-kube collect-parse --output my-collection.jsonl

  # Specify the parsed JSON output path
  bloodhound-kube collect-parse --parsed-output /tmp/bloodhound.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Use local log level if set, otherwise use global log level
		effectiveLogLevel := logLevel
		if !cmd.Flags().Changed("log") && globalLogLevel != "" {
			effectiveLogLevel = globalLogLevel
		}
		log := utils.New(effectiveLogLevel, globalNoColor)
		utils.SetDefaultLogger(log)

		jsonlPath, err := runCollect(cmd, args, log)
		if err != nil {
			return err
		}
		if jsonlPath == "" {
			return nil
		}

		clusterName := "unknown"
		if cluster, _ := cmd.Flags().GetString("cluster"); cluster != "" {
			clusterName = cluster
		}

		outputPath := resolveParsedOutputPath(jsonlPath, parsedOutput)
		return runParseFromFile(jsonlPath, outputPath, clusterName, log)
	},
}

func init() {
	addCollectFlags(collectParseCmd)
	collectParseCmd.Flags().StringVar(&parsedOutput, "parsed-output", "", "Output JSON file for parsed data (defaults to JSONL filename with .json extension)")
	collectParseCmd.Flags().String("cluster", "unknown", "Kubernetes cluster name for metadata")

	rootCmd.AddCommand(collectParseCmd)
}
