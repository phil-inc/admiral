package logstores

import (
	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/utils"
)

type Logstore interface {
	Init(c *config.Config) error
	Stream(ch chan utils.LogEntry)
}

type Default struct{}

func (d *Default) Init(c *config.Config) error {
	return nil
}

func (d *Default) Stream(ch chan utils.LogEntry) {}
