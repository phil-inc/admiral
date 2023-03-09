package main

import (
	"os"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/state"
	"github.com/phil-inc/admiral/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use: "admiral",
		Short: "Watch Kubernetes and stream to a backend",
		Long: `
		admiral is a set of Kubernetes controllers that will
		watch resources in the cluster and stream data to a
		backend.
		`,
		RunE: RootCmd,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				logrus.Printf("No argument(s) found -- starting up in monolith mode")
			}
			return cobra.OnlyValidArgs(cmd, args)
		},
		ValidArgs: []string{"events", "logs"},
	}
}

func RootCmd(cmd *cobra.Command, args []string) error {
	logrus.Println("Loading config...")
	path, err := cmd.Flags().GetString("path")
	if err != nil {
		return err
	}

	logrus.Println("\tOpening %s...", path)
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	logrus.Println("\tReading config...")
	cfg := config.Config{}
	err = cfg.Load(file)
	if err != nil {
		return err
	}
	file.Close()

	logrus.Println("Loaded config!")
	logrus.Println("")

	logrus.Println("Initializing shared mutable state...")
	s := state.New("")

	logrus.Println("\tAdding shared error channel to state...")
	errCh := make(chan error)
	s.SetErrChannel(errCh)
	go utils.HandleErrorStream(errCh)

	logrus.Println("\tAdding the kube client to state...")
	kubeClient, err := utils.GetClient()
	if err != nil {
		return err
	}
	s.SetKubeClient(kubeClient)

	logrus.Println("Initialized shared mutable state!")
	logrus.Println("")

	

	return nil
}
