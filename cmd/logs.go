package main

import (
	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var logsConfigCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream logs to a backend",
	Long: `
Open a watcher that filters pods and sends their logs to a remote logstore.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		// warn for too many arguments
		if len(args) > 0 {
			logrus.Warn("Unexpected argument(s) to command \"logs\". Expected 0 arguments.")
		}
		config := &config.Config{}
		if err := config.Load(); err != nil {
			logrus.Fatal(err)
		}
		client.Run(config, "logs")
	},
}

func init() {
	RootCmd.AddCommand(logsConfigCmd)
}
