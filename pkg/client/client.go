package client

import (
	"log"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/controller"
	"github.com/phil-inc/admiral/pkg/handlers"
	"github.com/phil-inc/admiral/pkg/handlers/webhook"
)

// Run runs the event loop on a given handler
func Run(conf *config.Config) {
	var eventHandler = ParseEventHandler(conf)
	controller.Start(conf, eventHandler)
}

// ParseEventHandler returns the first handler it finds (top to bottom)
func ParseEventHandler(conf *config.Config) handlers.Handler {
	var eventHandler handlers.Handler
	switch {
	case len(conf.Handler.Webhook.Url) > 0:
		eventHandler = new(webhook.Webhook)
	default:
		eventHandler = new(handlers.Default)
	}
	if err := eventHandler.Init(conf); err != nil {
		log.Fatal(err)
	}
	return eventHandler
}
