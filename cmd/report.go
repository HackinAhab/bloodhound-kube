package cmd

import (
	"fmt"
	"slices"
	"strings"

	"bloodhound-kube/internal/report"
	"bloodhound-kube/internal/utils"

	"github.com/spf13/cobra"
)

var (
	reportInputFile   string
	reportOutputFile  string
	reportType        string
	reportFormat      string
	reportVerbose     bool
	reportLogLevel    string
	trustedRegistries string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate security reports from collected Kubernetes resources",
	Long: `Generate security reports from JSONL files created by the collect command.

The report command analyzes Kubernetes resources and generates security-focused reports
broken down by type. It supports various report types including privileged containers,
security contexts, RBAC permissions, and more.

Examples:
  # Generate all security reports from collected data
  bloodhound-kube report -i data.jsonl --report all

  # Generate privileged containers report
  bloodhound-kube report -i data.jsonl --report privileged

  # Generate multiple specific reports (comma-delimited)
  bloodhound-kube report -i data.jsonl --report privileged,caps,imgsrc

  # Generate all reports with custom output prefix
  bloodhound-kube report -i data.jsonl --report all -o security-audit

  # Generate report in CSV format
  bloodhound-kube report -i data.jsonl --report caps --format csv

  # Generate image source report with trusted registries
  bloodhound-kube report -i data.jsonl --report imgsrc --trusted-registries registries.txt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := utils.New(reportLogLevel)

		if reportInputFile == "" {
			return fmt.Errorf("input file is required (-i/--input)")
		}

		// Parse and validate report types (comma-delimited)
		reportTypes := strings.Split(reportType, ",")
		for i, rt := range reportTypes {
			reportTypes[i] = strings.TrimSpace(rt)
		}

		validReportTypes := []string{"all", "privileged", "privesc", "nonroot", "caps", "imgsrc", "seccomp", "limits", "serviceaccount", "token"}
		for _, rt := range reportTypes {
			if !slices.Contains(validReportTypes, rt) {
				return fmt.Errorf("invalid report type %q, must be one of: %s", rt, strings.Join(validReportTypes, ", "))
			}
		}

		// Validate format
		if reportFormat != "json" && reportFormat != "csv" {
			return fmt.Errorf("invalid format %q, must be json or csv", reportFormat)
		}

		cfg := report.Config{
			InputFile:         reportInputFile,
			OutputPrefix:      reportOutputFile,
			ReportTypes:       reportTypes,
			Format:            reportFormat,
			TrustedRegistries: trustedRegistries,
			Verbose:           reportVerbose,
		}

		generator, err := report.NewGenerator(cfg, log)
		if err != nil {
			return fmt.Errorf("failed to create report generator: %w", err)
		}

		reports, err := generator.Generate()
		if err != nil {
			return fmt.Errorf("failed to generate reports: %w", err)
		}

		// Print summary
		for _, rep := range reports {
			if reportOutputFile != "" {
				ext := "json"
				if reportFormat == "csv" {
					ext = "csv"
				}
				filename := fmt.Sprintf("%s_%s.%s", reportOutputFile, rep.Type, ext)
				fmt.Printf("Generated %s report: %s (%d findings)\n", rep.Type, filename, rep.Count)
			} else {
				fmt.Printf("Generated %s report (%d findings)\n", rep.Type, rep.Count)
			}
		}

		return nil
	},
}

func init() {
	reportCmd.Flags().StringVarP(&reportInputFile, "input", "i", "", "Input JSONL file from collect command (required)")
	reportCmd.Flags().StringVarP(&reportOutputFile, "output", "o", "", "Output file prefix (optional, prints to stdout if not specified)")
	reportCmd.Flags().StringVar(&reportType, "report", "all", "Report type(s): all, privileged, privesc, nonroot, caps, imgsrc, seccomp, limits, serviceaccount, token (comma-delimited for multiple)")
	reportCmd.Flags().StringVar(&reportFormat, "format", "json", "Output format: json, csv")
	reportCmd.Flags().BoolVarP(&reportVerbose, "verbose", "v", false, "Verbose output")
	reportCmd.Flags().StringVarP(&reportLogLevel, "log", "l", "info", "Log level (debug, info, warn, error)")
	reportCmd.Flags().StringVar(&trustedRegistries, "trusted-registries", "", "File containing trusted registries (for imgsrc report)")

	reportCmd.MarkFlagRequired("input")
	rootCmd.AddCommand(reportCmd)
}
