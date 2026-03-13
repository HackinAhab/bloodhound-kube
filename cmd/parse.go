package cmd

import (
	"fmt"
	"os"

	"bloodhound-kube/internal/parser"
	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	inputFile           string
	outputFile          string
	parseLogLevel       string
	parseUndefinedNodes bool
)

func runParseFromFile(inputPath, outputPath, clusterName string, log utils.Logger, parseUndefinedNodes bool) error {
	if inputPath == "" {
		log.Error("Input file is required")
		return fmt.Errorf("input file is required")
	}

	log.Info("Reading input file", "file", inputPath)
	file, err := os.Open(inputPath)
	if err != nil {
		log.Error("Failed to open input file", "file", inputPath, "error", err)
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	if info, err := file.Stat(); err == nil {
		log.Debug("Successfully opened input file", "size", info.Size())
	}

	log.Debug("Using cluster name", "cluster", clusterName)

	log.Info("Parsing JSONL data")
	graph, err := parser.ConvertToBloodHoundResultFromReader(file, clusterName, parseUndefinedNodes)
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
		log, closeFn, err := buildLogger(effectiveLogLevel, true)
		if err != nil {
			return err
		}
		defer closeFn()
		utils.SetDefaultLogger(log)

		log.Debug("Starting parse command", "logLevel", effectiveLogLevel)

		// Determine cluster name
		clusterName := "unknown"
		if cluster, _ := cmd.Flags().GetString("cluster"); cluster != "" {
			clusterName = cluster
		}

		return runParseFromFile(inputFile, outputFile, clusterName, log, parseUndefinedNodes)
	},
}

func init() {
	parseCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input JSONL file from collect command")
	parseCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output JSON file (prints to stdout if not specified)")
	parseCmd.Flags().StringP("cluster", "c", "default-cluster", "Kubernetes cluster name for metadata")
	parseCmd.Flags().StringVarP(&parseLogLevel, "log", "l", "info", "Log level (trace, debug, info, warn, error)")
	parseCmd.Flags().BoolVar(&parseUndefinedNodes, "parse-undefined-nodes", false, "Enable generic node creation policy")

	rootCmd.AddCommand(parseCmd)
}
