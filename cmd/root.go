package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kube-bloodhound",
	Short: "A Kubernetes resource collector",
	Long:  "A CLI tool to collect and format Kubernetes resources into JSON",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize()
}
