package local

import (
	"github.com/phil-inc/admiral/config"
	"github.com/sirupsen/logrus"
)

type Local struct{}

func (l *Local) Init(c *config.Config) error {
	return nil
}

// Stream sends the logs to STDOUT
func (l *Local) Stream(log string, logMetadata map[string]string) error {
	logrus.Printf(log)
	return nil
}
