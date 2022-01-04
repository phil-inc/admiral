package loki

import (
	"fmt"
	"bytes"
	"net/http"
	"encoding/json"
	"time"

	"github.com/phil-inc/admiral/config"
)

type Loki struct {
	url string
}

type LokiDTO struct {
	Streams []Streams `json:"streams"`
}

type Streams struct {
	Stream map[string]string `json:"stream"`
	Values [][]string `json:"values"`
}

// Init creates the loki configuration
func (l *Loki) Init(c *config.Config) error {
	url := c.Logstream.Logstore.Loki.Url

	l.url = url

	return checkMissingVars(l)
}

// Stream sends the logs to Loki
func (l *Loki) Stream(log string, logMetadata map[string]string) error {
	msg := &LokiDTO{
		Streams: []Streams{
			{
				Stream: logMetadata,
				Values: [][]string{
					[]string{fmt.Sprintf("%d", time.Now().UnixNano()), log},
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

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func checkMissingVars(l *Loki) error {
	if l.url == "" {
		return fmt.Errorf("Loki URL not set")
	}

	return nil
}
