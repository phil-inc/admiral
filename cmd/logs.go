package main

import (
	"fmt"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Stream logs to a backend",
		Long: `
Open a watcher that filters pods and sends their logs to a remote logstore.
	`,
		RunE: LogsCmd,
	}
}

func LogsCmd(cmd *cobra.Command, args []string) error {
	// warn for too many arguments
	if len(args) > 0 {
		logrus.Warn("Unexpected argument(s) to command \"logs\". Expected 0 arguments.")
		return fmt.Errorf("%s", "Too many arguments")
	}
	config := &config.Config{}
	if err := config.Load(configPath); err != nil {
		return err
	}

	if err := client.Run(config, "logs"); err != nil {
		return err
	}

	return nil
}
