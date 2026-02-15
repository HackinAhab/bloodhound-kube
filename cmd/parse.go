package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bloodhound-kube/internal/parser"
	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	inputFile           string
	outputFile          string
	parseLogLevel       string
	policyDirs          string
	parseUndefinedNodes bool
)

func runParseFromFile(inputPath, outputPath, clusterName string, log utils.Logger, policyDirs []string, parseUndefinedNodes bool) error {
	if inputPath == "" {
		log.Error("Input file is required")
		return fmt.Errorf("input file is required")
	}

	log.Info("Reading input file", "file", inputPath)
	data, err := os.ReadFile(inputPath)
	if err != nil {
		log.Error("Failed to read input file", "file", inputPath, "error", err)
		return fmt.Errorf("failed to read input file: %w", err)
	}
	log.Debug("Successfully read input file", "size", len(data))

	log.Debug("Using cluster name", "cluster", clusterName)

	log.Info("Parsing JSONL data")
	resources, err := parser.ParseFromJSONL(data)
	if err != nil {
		return err
	}
	log.Debug("Successfully parsed JSONL", "resourceCount", len(resources))
	log.Info("Begin processing resources", "resourceCount", len(resources), "workers", 20)
	graph, err := parser.ConvertToBloodHoundResult(data, clusterName, policyDirs, parseUndefinedNodes)
	if err != nil {
		return err
	}
	log.Debug("Processing completed successfully")

	// Output as JSON
	log.Info("Marshaling result to JSON")
	jsonData, err := graph.ExportJSON(true)
	if err != nil {
		log.Error("Failed to marshal JSON", "error", err)
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	log.Debug("JSON marshaling completed", "size", len(jsonData))

	if outputPath != "" {
		log.Info("Writing output to file", "file", outputPath)
		if err := os.WriteFile(outputPath, []byte(jsonData), 0644); err != nil {
			log.Error("Failed to write output file", "file", outputPath, "error", err)
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("BloodHound Kubernetes data written to: %s\n", outputPath)

		nodeCount := 0
		edgeCount := 0
		if graph != nil {
			nodeCount = graph.GetNodeCount()
			edgeCount = graph.GetEdgeCount()
		}
		fmt.Printf("Processed %d nodes and %d edges from cluster: %s\n",
			nodeCount, edgeCount, clusterName)
		return nil
	}

	log.Debug("Writing output to stdout")
	fmt.Print(jsonData)
	return nil
}

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse collected resources to BloodHound format",
	Long: `Parse JSONL files from the collect command and convert them to BloodHound format.

This command takes the JSONL output from the collect command and transforms it into
BloodHound-compatible JSON format for graph analysis.

Examples:
  # Parse a collected file to BloodHound format
  bloodhound-kube parse -i collected-data.jsonl -o bloodhound-output.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Use local log level if set, otherwise use global log level
		effectiveLogLevel := parseLogLevel
		if !cmd.Flags().Changed("log") && globalLogLevel != "" {
			effectiveLogLevel = globalLogLevel
		}
		log := utils.New(effectiveLogLevel, globalNoColor)
		utils.SetDefaultLogger(log)

		log.Debug("Starting parse command", "logLevel", effectiveLogLevel)

		// Determine cluster name
		clusterName := "unknown"
		if cluster, _ := cmd.Flags().GetString("cluster"); cluster != "" {
			clusterName = cluster
		}

		return runParseFromFile(inputFile, outputFile, clusterName, log, parsePolicyDirs(policyDirs), parseUndefinedNodes)
	},
}

func parsePolicyDirs(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		paths = append(paths, filepath.Clean(trimmed))
	}

	return paths
}

func init() {
	parseCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input JSONL file from collect command")
	parseCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output JSON file (prints to stdout if not specified)")
	parseCmd.Flags().StringP("cluster", "c", "unknown", "Kubernetes cluster name for metadata")
	parseCmd.Flags().StringVarP(&parseLogLevel, "log", "l", "info", "Log level (debug, info, warn, error)")
	parseCmd.Flags().StringVar(&policyDirs, "policy-dirs", "", "Additional policy directories (comma-separated)")
	parseCmd.Flags().BoolVar(&parseUndefinedNodes, "parse-undefined-nodes", false, "Enable generic node creation policy")

	rootCmd.AddCommand(parseCmd)
}
