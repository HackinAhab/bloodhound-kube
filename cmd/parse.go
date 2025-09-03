package cmd

import (
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

		jsonData, err := bloodhound.ConvertToBloodHoundJSON(data)
		if err != nil {
			return fmt.Errorf("failed to parse data: %w", err)
		}

		if outputFile != "" {
			if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("BloodHound data written to: %s\n", outputFile)
		} else {
			fmt.Print(string(jsonData))
		}

		return nil
	},
}

func init() {
	parseCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input NDJSON file from collect command")
	parseCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output JSON file (prints to stdout if not specified)")
	parseCmd.Flags().Bool("stats", false, "Show parsing statistics")

	rootCmd.AddCommand(parseCmd)
}
