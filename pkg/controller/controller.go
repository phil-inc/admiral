package controller

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/handlers"
	"github.com/phil-inc/admiral/pkg/logstores"
	"github.com/phil-inc/admiral/pkg/utils"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var serverStartTime time.Time

// Start creates the informerFactory and initializes controllers
func Start(conf *config.Config, eventHandler handlers.Handler, logstore logstores.Logstore) {
	var kubeClient kubernetes.Interface

	if _, err := rest.InClusterConfig(); err != nil {
		kubeClient = utils.GetClientOutOfCluster()
	} else {
		kubeClient = utils.GetClient()
	}

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*30)

	serverStartTime = time.Now()

	eventCtrl := NewEventController(informerFactory, eventHandler, conf, kubeClient)
	eventStop := make(chan struct{})
	defer close(eventStop)
	err := eventCtrl.Run(eventStop)
	if err != nil {
		logrus.Fatal(err)
	}

	podCtrl := NewPodController(informerFactory, kubeClient, conf, logstore)
	podStop := make(chan struct{})
	defer close(podStop)
	err = podCtrl.Run(podStop)
	if err != nil {
		logrus.Fatal(err)
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}
