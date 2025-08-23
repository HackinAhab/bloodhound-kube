package main

import (
	"os"

	"kube-bloodhound/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}