package main

import (
	"fmt"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewEventsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "events",
		Short: "Stream events to a backend",
		Long: `
	Open a watcher that filters events and sends their text to a remote backend.
		`,
		RunE: EventsCmd,
	}
}

func EventsCmd(cmd *cobra.Command, args []string) error {
	// warn for too many arguments
	if len(args) > 0 {
		logrus.Warn("Unexpected argument(s) to command \"events\". Expected 0 arguments.")
		return fmt.Errorf("Too many arguments")
	}
	config := &config.Config{}

	if err := config.Load(configPath); err != nil {
		return err
	}

	if err := client.Run(config, "events"); err != nil {
		return err
	}

	return nil
}
