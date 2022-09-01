package loki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/utils"
	"github.com/sirupsen/logrus"
)

type Loki struct {
	url    string
	client *http.Client
}

type LokiDTO struct {
	Streams []Streams `json:"streams"`
}

type Streams struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// Init creates the loki configuration
func (l *Loki) Init(c *config.Config) error {
	url := c.Logstream.Logstore.Loki.Url

	l.url = url

	l.client = &http.Client{}

	return checkMissingVars(l)
}

// Stream sends the logs to Loki
func (l *Loki) Stream(entry chan utils.LogEntry) {
	for {
		select {
		case e := <-entry:
			err := l.Send(e.Text, e.Metadata)
			if err != nil {
				logrus.Error(err)
			}
		}
	}
}

func (l *Loki) Send(log string, metadata map[string]string) error {
	msg := &LokiDTO{
		Streams: []Streams{
			{
				Stream: metadata,
				Values: [][]string{
					{log},
				},
			},
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/loki/api/v1/push", l.url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := l.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 204 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		logrus.Printf("%s - %s", res.Status, buf.String())
	}

	return nil
}

func checkMissingVars(l *Loki) error {
	if l.url == "" {
		return fmt.Errorf("Loki URL not set")
	}

	return nil
}
