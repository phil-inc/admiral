package metrics_handlers

import (
	"github.com/phil-inc/admiral/config"
)

type MetricsHandler interface {
	Init(c *config.Config) error
	Handle(metrics <-chan MetricBatch)
}

type Default struct{}

func (d *Default) Init(c *config.Config) error {
	return nil
}

func (d *Default) Handle(metrics <-chan MetricBatch) {}
