package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/backend"
	"github.com/phil-inc/admiral/pkg/state"
	"github.com/phil-inc/admiral/pkg/utils"
	"github.com/phil-inc/admiral/pkg/watcher/events"
	"github.com/phil-inc/admiral/pkg/watcher/logs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/informers"
)

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "admiral",
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
				logrus.Println("")
			}
			return cobra.OnlyValidArgs(cmd, args)
		},
		ValidArgs: []string{"events", "logs"},
	}
}

func RootCmd(cmd *cobra.Command, args []string) error {
	logrus.Println("Loading config...")
	path, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	logrus.Printf("\tOpening %s...", path)
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

	logrus.Println("Initializing kube informer factory...")

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*30)

	logrus.Println("\tInitializing watchers...")

	rawLogCh := make(chan backend.RawLog)
	eventCh := make(chan string)

	httpCli := &http.Client{}

	for _, w := range cfg.Watchers {

		switch w.Type {

		case "logs":
			l := logs.New().State(s).PodFilterAnnotation(w.PodFilterAnnotation).IgnoreContainerAnnotation(w.IgnoreContainerAnnotation).RawLogChannel(rawLogCh).Build()

			podInformer := informerFactory.Core().V1().Pods()

			logrus.Println("\t\tLog informer created")

			err = InitBackend(rawLogCh, nil, errCh, httpCli, w.Backend.Type, w.Backend.URL)
			if err != nil {
				return err
			}

			logrus.Printf("\t\t%s backend initialized", w.Backend.Type)

			err = InitWatcher(l, podInformer.Informer())
			if err != nil {
				return err
			}

			logrus.Println("\t\tLog informer initialized")
			logrus.Println("")

		case "events":
			e := events.New().State(s).Filter(w.Filter).Channel(eventCh).Build()

			eventInformer := informerFactory.Core().V1().Events()

			logrus.Println("\t\tEvent informer created")

			err = InitBackend(nil, eventCh, errCh, httpCli, w.Backend.Type, w.Backend.URL)
			if err != nil {
				return err
			}

			logrus.Printf("\t\t%s backend initialized", w.Backend.Type)

			err = InitWatcher(e, eventInformer.Informer())
			if err != nil {
				return err
			}

			logrus.Println("\t\tLog informer initialized")
			logrus.Println("")

		default:
			return errors.Errorf("invalid type in watcher: %s", w.Type)

		}
	}

	logrus.Println("Watchers: Initialized")
	logrus.Println("Backends: Initialized")

	stop := make(chan struct{})
	defer close(stop)

	informerFactory.Start(stop)
	logrus.Println("Admiral: Ready")
	logrus.Println("")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm

	return nil
}
