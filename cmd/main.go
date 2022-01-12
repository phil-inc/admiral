package main

import (
	"fmt"
	"os"

	"github.com/common-nighthawk/go-figure"
	"github.com/phil-inc/admiral/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const admiralConfigFile = ".admiral.yaml"

var philLogo = figure.NewColorFigure("phil, inc.", "", "cyan", true)
var admiralLogo = figure.NewColorFigure("Admiral", "", "green", true)
var cfgFile string

var RootCmd = &cobra.Command{
	Use:   "admiral",
	Short: "A controller managing Kubernetes",
	Long: `
Admiral: A controller for managing Kubernetes operations

Admiral is a series of controllers integrating across a Kubernetes
cluster to do operations on behalf of the operator.
`,

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			logrus.Warn("Unexpected argument(s) to command \"admiral\". Expected 0 arguments.")
		}
		config := &config.Config{}
		if err := config.Load(); err != nil {
			logrus.Fatal(err)
		}
		logrus.Info("See \"admiral help\" for information on how to use Admiral")
	},
}

// Execute adds child commands to the root command
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.SetHelpCommand(&cobra.Command{
		Use:    "help",
		Hidden: true,
		Short:  "Admiral automates operations in a Kubernetes cluster",
		Long: `
	Find more information at https://github.com/philinc/admiral

Commands:
	events	Stream events from a cluster to a backend
	logs	Stream logs from a cluster to a backend
		`,
	})
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(admiralConfigFile)
	viper.AddConfigPath("$HOME")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	admiralLogo.Print()
	philLogo.Print()
	Execute()
}
