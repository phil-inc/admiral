package local

import (
	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/utils"
	"github.com/sirupsen/logrus"
)

type Local struct{}

func (l *Local) Init(c *config.Config) error {
	return nil
}

// Stream sends the logs to STDOUT
func (l *Local) Stream(entry chan utils.LogEntry) {
	for {
		select {
		case e := <-entry:
			l.Send(e.Text, e.Metadata)
		}
	}
}

func (l *Local) Send(log string, metadata map[string]string) {
	logrus.Printf(log)
}
