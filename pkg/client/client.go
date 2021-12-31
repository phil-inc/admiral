package client

import (
	"log"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/controller"
	"github.com/phil-inc/admiral/pkg/handlers"
	"github.com/phil-inc/admiral/pkg/handlers/webhook"
	"github.com/phil-inc/admiral/pkg/logstores"
	"github.com/phil-inc/admiral/pkg/logstores/loki"
)

// Run runs the event loop on a given handler
func Run(conf *config.Config) {
	var eventHandler = ParseEventHandler(conf)
	var logStore = ParseLogHandler(conf)
	controller.Start(conf, eventHandler, logStore)
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
	default:
		logHandler = new(logstores.Default)
	}
	if err := logHandler.Init(conf); err != nil {
		log.Fatal(err)
	}
	return logHandler
}
