package cmd

import (
	"fmt"
	"os"

	"bloodhound-kube/internal/parser"
	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	inputFile     string
	outputFile    string
	parseLogLevel string
)

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
		log := utils.New(effectiveLogLevel)

		log.Debug("Starting parse command", "logLevel", effectiveLogLevel)

		if inputFile == "" {
			log.Error("Input file is required")
			return fmt.Errorf("input file is required")
		}

		log.Info("Reading input file", "file", inputFile)
		data, err := os.ReadFile(inputFile)
		if err != nil {
			log.Error("Failed to read input file", "file", inputFile, "error", err)
			return fmt.Errorf("failed to read input file: %w", err)
		}
		log.Debug("Successfully read input file", "size", len(data))

		// Determine cluster name
		clusterName := "unknown"
		if cluster, _ := cmd.Flags().GetString("cluster"); cluster != "" {
			clusterName = cluster
		}
		log.Debug("Using cluster name", "cluster", clusterName)

		log.Info("Parsing JSONL data")
		resources, err := parser.ParseFromJSONL(data)
		if err != nil {
			log.Error("Failed to parse JSONL", "error", err)
			return fmt.Errorf("failed to parse JSONL: %w", err)
		}
		log.Debug("Successfully parsed JSONL", "resourceCount", len(resources))
		log.Info("Begin processing resources", "resourceCount", len(resources), "workers", 20)
		graph, err := parser.ConvertToBloodHoundResult(data, clusterName)
		if err != nil {
			log.Error("Failed to process data concurrently", "error", err)
			return fmt.Errorf("failed to process data concurrently: %w", err)
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

		if outputFile != "" {
			log.Info("Writing output to file", "file", outputFile)
			if err := os.WriteFile(outputFile, []byte(jsonData), 0644); err != nil {
				log.Error("Failed to write output file", "file", outputFile, "error", err)
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("BloodHound-compliant data written to: %s\n", outputFile)

			nodeCount := 0
			edgeCount := 0
			if graph != nil {
				nodeCount = graph.GetNodeCount()
				edgeCount = graph.GetEdgeCount()
			}
			fmt.Printf("Processed %d nodes and %d edges from cluster: %s\n",
				nodeCount, edgeCount, clusterName)
		} else {
			log.Debug("Writing output to stdout")
			fmt.Print(jsonData)
		}
		return nil
	},
}

func init() {
	parseCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input JSONL file from collect command")
	parseCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output JSON file (prints to stdout if not specified)")
	parseCmd.Flags().StringP("cluster", "c", "unknown", "Kubernetes cluster name for metadata")
	parseCmd.Flags().StringVarP(&parseLogLevel, "log", "l", "info", "Log level (debug, info, warn, error)")

	rootCmd.AddCommand(parseCmd)
}
