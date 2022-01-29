package main

import(
	"fmt"
	"github.com/spf13/cobra"
	"github.com/sirupsen/logrus"
)

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "admiral",
		Short: "A controller managing Kubernetes",
		Long: `
	Admiral: A controller for managing Kubernetes operations
	
	Admiral is a series of controllers integrating across a Kubernetes
	cluster to do operations on behalf of the operator.
	`,
		RunE: RootCmd,
	}
}

func RootCmd(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		logrus.Warn("Unexpected argument(s) to command \"admiral\". Expected 0 arguments.")
		return fmt.Errorf("%s", "Too many arguments")
	}
	logrus.Printf("See \"admiral help\" for information on how to use Admiral")
	return nil
}