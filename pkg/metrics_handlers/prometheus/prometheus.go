package prometheus

import "github.com/phil-inc/admiral/config"

type Prometheus struct{}

// Init binds any settings from the config to the handler
func (p *Prometheus) Init(c *config.Config) error {
	return nil
}

// Handle exposes the metrics for Prometheus scraping
func (p *Prometheus) Handle() error {
	return nil
}
