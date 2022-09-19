package loki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

	l.client = &http.Client{
		Timeout: 5 * time.Second,
	}

	return checkMissingVars(l)
}

// Stream sends the logs to Loki
func (l *Loki) Stream(entry chan utils.LogEntry) {
	for e := range entry {
		if e.Err == nil {
			l.Send(e.Text, e.Metadata)
		}
	}
}

func (l *Loki) Send(log string, metadata map[string]string) {

	timeStamp := time.Now().UnixNano()

	if log[0:1] == "{" {
		// potentially a json log. parse to get all labels
		var logLine map[string]string

		err := json.Unmarshal([]byte(log), &logLine)
		if err == nil {
			for k, v := range logLine {

				if k == "msg" {
					log = v
					continue
				}

				if k == "time" {
					t, err := time.Parse(time.RFC3339, v)
					if err == nil {
						timeStamp = t.UnixNano()
					}
					continue
				}

				metadata[k] = v
			}
		}
	}

	msg := &LokiDTO{
		Streams: []Streams{
			{
				Stream: metadata,
				Values: [][]string{
					{fmt.Sprintf("%d", timeStamp), log},
				},
			},
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		logrus.Error(err)
	}

	url := fmt.Sprintf("%s/loki/api/v1/push", l.url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		logrus.Error(err)
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := l.client.Do(req)
	if err != nil {
		logrus.Error(err)
	} else if res.StatusCode != 204 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		logrus.Errorf("%s - %s", res.Status, buf.String())
	}
}

func checkMissingVars(l *Loki) error {
	if l.url == "" {
		return fmt.Errorf("Loki URL not set")
	}

	return nil
}
