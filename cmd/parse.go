package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"bloodhound-kube/internal/bloodhound"
	"bloodhound-kube/internal/bloodhound/nodes"

	"github.com/spf13/cobra"
)

var (
	inputFile  string
	outputFile string
)

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse collected resources to BloodHound format",
	Long: `Parse NDJSON files from the collect command and convert them to BloodHound format.

This command takes the NDJSON output from the collect command and transforms it into
BloodHound-compatible JSON format for graph analysis.

Examples:
  # Parse a collected file to BloodHound format
  bloodhound-kube parse -i collected-data.ndjson -o bloodhound-output.json

  # Show parsing statistics
  bloodhound-kube parse --stats`,
	RunE: func(cmd *cobra.Command, args []string) error {
		nodes.RegisterParsers()

		if cmd.Flags().Changed("stats") {
			bloodhound.PrintParsingStats()
			return nil
		}

		if inputFile == "" {
			return fmt.Errorf("input file is required")
		}

		data, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read input file: %w", err)
		}

		// Determine cluster name
		clusterName := "unknown"
		if cluster, _ := cmd.Flags().GetString("cluster"); cluster != "" {
			clusterName = cluster
		}

		// Use concurrent processing for better performance on large datasets
		resources, err := bloodhound.ParseFromNDJSON(data)
		if err != nil {
			return fmt.Errorf("failed to parse NDJSON: %w", err)
		}

		// Use concurrent processing if we have a large number of resources
		var result *bloodhound.BloodHoundResult
		if len(resources) > 1000 {
			// Use concurrent processing for large datasets
			result, err = bloodhound.ConcurrentParseProcessor(resources, 20) // 20 workers for high concurrency
			if err != nil {
				return fmt.Errorf("failed to process data concurrently: %w", err)
			}
			// Set cluster name in metadata
			result.Metadata.ClusterName = clusterName
		} else {
			// Use regular processing for smaller datasets
			result, err = bloodhound.ConvertToBloodHoundResult(data, clusterName)
			if err != nil {
				return fmt.Errorf("failed to convert data: %w", err)
			}
		}

		// Output as JSON
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		if outputFile != "" {
			if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("BloodHound-compliant data written to: %s\n", outputFile)
			fmt.Printf("Processed %d nodes and %d edges from cluster: %s\n",
				len(result.Graph.Nodes), len(result.Graph.Edges), clusterName)
		} else {
			fmt.Print(string(jsonData))
		}

		return nil
	},
}

func init() {
	parseCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input NDJSON file from collect command")
	parseCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output JSON file (prints to stdout if not specified)")
	parseCmd.Flags().StringP("cluster", "c", "unknown", "Kubernetes cluster name for metadata")
	parseCmd.Flags().Bool("stats", false, "Show parsing statistics")

	rootCmd.AddCommand(parseCmd)
}
