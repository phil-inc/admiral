package main

import (
	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var eventsConfigCmd = &cobra.Command{
	Use:   "events",
	Short: "Stream events to a backend",
	Long: `
Open a watcher that filters events and sends their text to a remote backend.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		// warn for too many arguments
		if len(args) > 0 {
			logrus.Warn("Unexpected argument(s) to command \"events\". Expected 0 arguments.")
		}
		config := &config.Config{}
		if err := config.Load(); err != nil {
			logrus.Fatal(err)
		}
		client.Run(config, "events")
	},
}

func init() {
	RootCmd.AddCommand(eventsConfigCmd)
}
