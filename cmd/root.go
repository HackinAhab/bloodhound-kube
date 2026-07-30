package cmd

import (
	"bloodhound-kube/internal/utils"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Version is stamped at build time via -ldflags "-X bloodhound-kube/cmd.Version=...".
var Version = "dev"

// versionString appends build tags (read from the binary's embedded build
// info) to Version, e.g. "dev (tags: embedded,no_calico)".
func versionString() string {
	v := Version
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "-tags" && s.Value != "" {
				v = fmt.Sprintf("%s (tags: %s)", v, s.Value)
				break
			}
		}
	}
	return v
}

var (
	globalLogLevel string
	globalNoColor  bool
	globalLogFile  string
)

func buildLogger(level string, includeFile bool) (*utils.Logger, func(), error) {
	var file *os.File
	output := io.Writer(os.Stderr)
	if includeFile && globalLogFile != "" {
		opened, err := os.OpenFile(globalLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file %s: %w", globalLogFile, err)
		}
		file = opened
		output = io.MultiWriter(os.Stderr, file)
	}

	logger := utils.NewWithOutput(level, globalNoColor, output)
	closeFn := func() {
		if file != nil {
			_ = file.Close()
		}
	}

	return logger, closeFn, nil
}

var rootCmd = &cobra.Command{
	Use:           "bloodhound-kube",
	Version:       versionString(),
	Short:         "A Kubernetes resource collector for Bloodhound",
	Long:          "A CLI tool to collect and format Kubernetes resources into Bloodhound OpenGraph compatible JSON",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level := globalLogLevel
		if level == "" {
			level = "info"
		}
		logger, _, err := buildLogger(level, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to configure logger: %v\n", err)
			utils.SetDefaultLogger(utils.New(level, globalNoColor))
			return
		}
		utils.SetDefaultLogger(logger)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize()
	rootCmd.PersistentFlags().StringVarP(&globalLogLevel, "log", "l", "info", "Log level (trace, debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&globalNoColor, "no-color", false, "Disable colored log output")
	rootCmd.PersistentFlags().StringVar(&globalLogFile, "log-file", "", "Write logs to file in addition to stderr")
}
