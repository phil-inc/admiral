package main

import (
	"fmt"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewPerformanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "performance",
		Short: "Run performance testing",
		Long:  `Open a watcher that runs performance testing.`,
		RunE:  PerformanceCmd,
	}
}

func PerformanceCmd(cmd *cobra.Command, args []string) error {
	// warn for too many arguments
	if len(args) > 0 {
		logrus.Warn("Unexpected argument(s) to command \"performance\". Expected 0 arguments.")
		return fmt.Errorf("%s", "Too many arguments")
	}
	config := &config.Config{}
	if err := config.Load(configPath); err != nil {
		return err
	}

	if err := client.Run(config, "performance"); err != nil {
		return err
	}

	return nil
}
