package handlers

import (
	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/event"
)

type Handler interface {
	Init(c *config.Config) error
	Handle(e event.Event)
}

type Default struct{}

func (d *Default) Init(c *config.Config) error {
	return nil
}

func (d *Default) Handle(e event.Event) {}
