package logstores

import (
	"github.com/phil-inc/admiral/config"
)

type Logstore interface {
	Init(c *config.Config) error
	Stream(log string, pod string, container string) error
}

type Default struct{}

func (d *Default) Init(c *config.Config) error {
	return nil
}

func (d *Default) Stream(log string, pod string, container string) error {
	return nil
}
