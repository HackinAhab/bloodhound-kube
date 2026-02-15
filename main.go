package main

import (
	"os"

	"bloodhound-kube/cmd"
	"bloodhound-kube/internal/utils"
)

func main() {
	if err := cmd.Execute(); err != nil {
		utils.DefaultLogger().Error("Command failed", "error", err)
		os.Exit(1)
	}
}
