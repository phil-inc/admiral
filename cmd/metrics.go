package main

import (
	"fmt"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewMetricsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "metrics",
		Short: "Stream metrics to a backend",
		Long: `
Open a watcher that filters pods and sends their metrics to a remote backend.
	`,
		RunE: MetricsCmd,
	}
}

func MetricsCmd(cmd *cobra.Command, args []string) error {
	// warn for too many arguments
	if len(args) > 0 {
		logrus.Warn("Unexpected argument(s) to command \"metrics\". Expected 0 arguments.")
		return fmt.Errorf("%s", "Too many arguments")
	}
	config := &config.Config{}
	if err := config.Load(configPath); err != nil {
		return err
	}

	if err := client.Run(config, "metrics"); err != nil {
		return err
	}

	return nil
}
