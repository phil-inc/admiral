package main

import (
	"os"

	"github.com/common-nighthawk/go-figure"
	"github.com/sirupsen/logrus"
)

var philLogo = figure.NewColorFigure("phil, inc.", "", "cyan", true)
var admiralLogo = figure.NewColorFigure("Admiral", "", "green", true)
var configPath string

func main() {
	admiralLogo.Print()
	philLogo.Print()

	rootCmd := NewRootCmd()
	rootCmd.SetHelpCommand(NewHelpCmd())
	rootCmd.AddCommand(
		NewLogsCmd(),
		NewEventsCmd(),
		NewPerformanceCmd(),
	)
	rootCmd.PersistentFlags().StringVarP(&configPath, "file", "f", "", "specify a path to a YAML file")

	err := rootCmd.Execute()
	if err != nil {
		logrus.Error(err)
		os.Exit(-1)
	}
}
