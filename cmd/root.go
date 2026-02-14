package cmd

import (
	"github.com/spf13/cobra"
)

var (
	globalLogLevel string
	globalNoColor  bool
)

var rootCmd = &cobra.Command{
	Use:   "bloodhound-kube",
	Short: "A Kubernetes resource collector for Bloodhound",
	Long:  "A CLI tool to collect and format Kubernetes resources into Bloodhound OpenGraph compatible JSON",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize()
	rootCmd.PersistentFlags().StringVarP(&globalLogLevel, "log", "l", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&globalNoColor, "no-color", false, "Disable colored log output")
}
