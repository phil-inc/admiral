package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/sirupsen/logrus"
	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/event"
)

type Webhook struct {
	url string
}

type WebhookMessage struct {
	Text string `json:"text"`
}

// Init creates the webhook configuration
func (w *Webhook) Init(c *config.Config) error {
	url := c.Events.Handler.Webhook.Url

	w.url = url

	return checkMissingVars(w)
}

func (w *Webhook) Handle(e event.Event) {
	msg := prepareWebhookMessage(e, w)
	err := postMessage(w.url, msg)
	if err != nil {
		logrus.Errorf("%s\n", err)
		return
	}

	logrus.Printf("Message sent: %s", msg)
}

func checkMissingVars(w *Webhook) error {
	if w.url == "" {
		return fmt.Errorf("Webhook URL not set")
	}

	return nil
}

func prepareWebhookMessage(e event.Event, w *Webhook) *WebhookMessage {
	return &WebhookMessage{
		Text: e.Message(),
	}
}

func postMessage(url string, msg *WebhookMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

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
