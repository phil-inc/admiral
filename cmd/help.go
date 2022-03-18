package main

import (
	"github.com/spf13/cobra"
)

func NewHelpCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "help",
		Hidden: true,
		Short:  "Admiral automates operations in a Kubernetes cluster",
		Long: `
Find more information at https://github.com/philinc/admiral

Commands:
events	Stream events from a cluster to a backend
logs	Stream logs from a cluster to a backend
metrics	Stream metrics from the cluster to a backend
	`,
	}
}
