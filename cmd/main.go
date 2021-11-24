package main

import (
	"fmt"
	"os"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const admiralConfigFile = ".admiral.yaml"

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
		config := &config.Config{}
		if err := config.Load(); err != nil {
			logrus.Fatal(err)
		}
		client.Run(config)
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

	// Disable Help subcommand
	RootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
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
	Execute()
}
