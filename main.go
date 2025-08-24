package main

import (
	"os"

	"bloodhound-kube/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
