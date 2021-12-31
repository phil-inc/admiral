package logstores

import "github.com/phil-inc/admiral/config"

type Logstore interface {
	Init(c *config.Config) error
}

type Default struct{}

func (d *Default) Init(c *config.Config) error {
	return nil
}
