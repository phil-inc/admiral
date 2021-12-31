package loki

import (
	"fmt"

	"github.com/phil-inc/admiral/config"
)

type Loki struct {
	url string
}

// Init creates the loki configuration
func (l *Loki) Init(c *config.Config) error {
	url := c.Logstream.Logstore.Loki.Url

	l.url = url

	return checkMissingVars(l)
}

func checkMissingVars(l *Loki) error {
	if l.url == "" {
		return fmt.Errorf("Loki URL not set")
	}

	return nil
}
