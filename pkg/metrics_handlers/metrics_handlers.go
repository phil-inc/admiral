package metrics_handlers

import (
	"github.com/phil-inc/admiral/config"
)

type MetricsHandler interface {
	Init(c *config.Config) error
}

type Default struct{}

func (d *Default) Init(c *config.Config) error {
	return nil
}

func (d *Default) Handle() error {
	return nil
}
