package client

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/controllers"
	"github.com/phil-inc/admiral/pkg/controllers/events"
	"github.com/phil-inc/admiral/pkg/controllers/logs"
	"github.com/phil-inc/admiral/pkg/controllers/metrics"
	"github.com/phil-inc/admiral/pkg/controllers/performance"
	"github.com/phil-inc/admiral/pkg/handlers"
	"github.com/phil-inc/admiral/pkg/handlers/webhook"
	"github.com/phil-inc/admiral/pkg/logstores"
	"github.com/phil-inc/admiral/pkg/logstores/local"
	"github.com/phil-inc/admiral/pkg/logstores/loki"
	"github.com/phil-inc/admiral/pkg/metrics_handlers"
	"github.com/phil-inc/admiral/pkg/metrics_handlers/prometheus"
	"github.com/phil-inc/admiral/pkg/target"
	"github.com/phil-inc/admiral/pkg/target/web"
	"github.com/phil-inc/admiral/pkg/utils"
	"k8s.io/client-go/informers"
)

// Run runs the event loop on a given handler
func Run(conf *config.Config, operation string) error {
	kubeClient, err := utils.GetClient()
	if err != nil {
		return err
	}

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*30)

	var ctrl controllers.Controller
	switch operation {
	case "logs":
		var logStore = ParseLogHandler(conf)
		ctrl = logs.NewLogController(informerFactory, kubeClient, conf, logStore)
	case "events":
		var eventHandler = ParseEventHandler(conf)
		ctrl = events.NewEventController(informerFactory, eventHandler, conf, kubeClient)
	case "performance":
		var target = ParsePerformanceHandler(conf)
		ctrl = performance.NewPerformanceController(informerFactory, kubeClient, conf, target)
	case "metrics":
		var metricsHandler = ParseMetricsHandler(conf)
		ctrl = metrics.NewMetricsController(informerFactory, metricsHandler, conf)
	}

	ctrlStop := ctrl.Watch()
	defer close(ctrlStop)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm

	return nil
}

// ParseEventHandler returns the first event handler it finds (top to bottom)
func ParseEventHandler(conf *config.Config) handlers.Handler {
	var eventHandler handlers.Handler
	switch {
	case len(conf.Events.Handler.Webhook.Url) > 0:
		eventHandler = new(webhook.Webhook)
	default:
		eventHandler = new(handlers.Default)
	}
	if err := eventHandler.Init(conf); err != nil {
		log.Fatal(err)
	}
	return eventHandler
}

// ParseLogHandler returns the first logstream handler it finds (top to bottom)
func ParseLogHandler(conf *config.Config) logstores.Logstore {
	var logHandler logstores.Logstore
	switch {
	case len(conf.Logstream.Logstore.Loki.Url) > 0:
		logHandler = new(loki.Loki)
	case conf.Logstream.Logstore.Local:
		logHandler = new(local.Local)
	default:
		logHandler = new(logstores.Default)
	}
	if err := logHandler.Init(conf); err != nil {
		log.Fatal(err)
	}
	return logHandler
}

// ParsePerformanceHandler returns the first result handler it finds (top to bottom)
func ParsePerformanceHandler(conf *config.Config) target.Target {
	var target target.Target
	switch {
	case len(conf.Performance.Target.Web.Tests) > 0:
		target = new(web.Web)
	}

	if err := target.Init(conf); err != nil {
		log.Fatal(err)
	}
	return target
}

// ParseMetricsHandler returns the first metrics handler it finds (top to bottom)
func ParseMetricsHandler(conf *config.Config) metrics_handlers.MetricsHandler {
	var metricsHandler metrics_handlers.MetricsHandler
	switch {
	case len(conf.Metrics.Handler.Prometheus) > 0:
		metricsHandler = new(prometheus.Prometheus)
	default:
		metricsHandler = new(metrics_handlers.Default)
	}
	if err := metricsHandler.Init(conf); err != nil {
		log.Fatal(err)
	}
	return metricsHandler
}
