package cmd

import (
	"fmt"
	"os"

	"bloodhound-kube/internal/parser"
	"bloodhound-kube/internal/utils"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	inputFile     string
	outputFile    string
	parseLogLevel string
)

func runParseFromFile(inputPath, outputPath, clusterName string, log logrus.FieldLogger) error {
	if inputPath == "" {
		log.Error("Input file is required")
		return fmt.Errorf("input file is required")
	}

	log.WithField("file", inputPath).Info("Reading input file")
	data, err := os.ReadFile(inputPath)
	if err != nil {
		log.WithError(err).WithField("file", inputPath).Error("Failed to read input file")
		return fmt.Errorf("failed to read input file: %w", err)
	}
	log.WithField("size", len(data)).Debug("Successfully read input file")

	log.WithField("cluster", clusterName).Debug("Using cluster name")

	log.Info("Parsing JSONL data")
	resources, err := parser.ParseFromJSONL(data)
	if err != nil {
		log.WithError(err).Error("Failed to parse JSONL")
		return fmt.Errorf("failed to parse JSONL: %w", err)
	}
	log.WithField("resourceCount", len(resources)).Debug("Successfully parsed JSONL")
	log.WithFields(logrus.Fields{"resourceCount": len(resources), "workers": 20}).Info("Begin processing resources")
	graph, err := parser.ConvertToBloodHoundResult(data, clusterName)
	if err != nil {
		log.WithError(err).Error("Failed to process data concurrently")
		return fmt.Errorf("failed to process data concurrently: %w", err)
	}
	log.Debug("Processing completed successfully")

	// Output as JSON
	log.Info("Marshaling result to JSON")
	jsonData, err := graph.ExportJSON(true)
	if err != nil {
		log.WithError(err).Error("Failed to marshal JSON")
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	log.WithField("size", len(jsonData)).Debug("JSON marshaling completed")

	if outputPath != "" {
		log.WithField("file", outputPath).Info("Writing output to file")
		if err := os.WriteFile(outputPath, []byte(jsonData), 0644); err != nil {
			log.WithError(err).WithField("file", outputPath).Error("Failed to write output file")
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

		log.WithField("logLevel", effectiveLogLevel).Debug("Starting parse command")

		// Determine cluster name
		clusterName := "unknown"
		if cluster, _ := cmd.Flags().GetString("cluster"); cluster != "" {
			clusterName = cluster
		}

		return runParseFromFile(inputFile, outputFile, clusterName, log)
	},
}

func init() {
	parseCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input JSONL file from collect command")
	parseCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output JSON file (prints to stdout if not specified)")
	parseCmd.Flags().StringP("cluster", "c", "unknown", "Kubernetes cluster name for metadata")
	parseCmd.Flags().StringVarP(&parseLogLevel, "log", "l", "info", "Log level (debug, info, warn, error)")

	rootCmd.AddCommand(parseCmd)
}
