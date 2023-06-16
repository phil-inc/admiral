package main

import "github.com/spf13/cobra"

func NewHelpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "help",
		Short: "admiral watches Kubernetes resources and streams to a backend",
		Long:  "Find more information at https://github.com/phil-inc/admiral",
	}
}
