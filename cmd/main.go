package main

import (
	"os"

	"github.com/common-nighthawk/go-figure"
	"github.com/sirupsen/logrus"
)

var philLogo = figure.NewColorFigure("phil, inc.", "", "cyan", true)
var admiralLogo = figure.NewColorFigure("Admiral", "", "green", true)

func main() {
	admiralLogo.Print()
	philLogo.Print()

	rootCmd := NewRootCmd()
	rootCmd.SetHelpCommand(NewHelpCmd())
	rootCmd.PersistentFlags().String("config", "/admiral.yaml", "A path to a config file")

	err := rootCmd.Execute()
	if err != nil {
		logrus.Error(err)
		os.Exit(-1)
	}
}
