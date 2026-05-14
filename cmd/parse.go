package cmd

import (
	"bloodhound-kube/internal/cli"
	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	inputFile           string
	outputFile          string
	parseLogLevel       string
	parseUndefinedNodes bool
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

		_, err = cli.ParseService{}.Run(cli.ParseRequest{
			InputPath:           inputFile,
			OutputPath:          outputFile,
			ClusterName:         clusterName,
			ParseUndefinedNodes: parseUndefinedNodes,
		}, log)
		return err
	},
}

func init() {
	parseCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input JSONL file from collect command")
	parseCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output JSON file (prints to stdout if not specified)")
	parseCmd.Flags().StringP("cluster", "c", "default", "Kubernetes cluster name for metadata")
	parseCmd.Flags().StringVarP(&parseLogLevel, "log", "l", "info", "Log level (trace, debug, info, warn, error)")
	parseCmd.Flags().BoolVar(&parseUndefinedNodes, "parse-undefined-nodes", false, "Enable generic node creation policy")

	rootCmd.AddCommand(parseCmd)
}
