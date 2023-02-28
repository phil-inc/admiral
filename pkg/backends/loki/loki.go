package loki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Builder struct {
	url        string
	client     *http.Client
	logChannel chan string
	errChannel chan error
}

// New returns a Builder for the Loki struct.
func New() *Builder {
	return &Builder{}
}

// Build returns a configured Loki struct.
func (b *Builder) Build() *loki {
	return &loki{
		url:        b.url,
		client:     b.client,
		logChannel: b.logChannel,
		errChannel: b.errChannel,
	}
}

// Url takes the hostname of the Loki instance
// and builds the full log-pushing API url.
func (b *Builder) Url(u string) *Builder {
	b.url = fmt.Sprintf("%s/loki/api/v1/push", u)
	return b
}

// Client injects an HTTP client.
func (b *Builder) Client(cli *http.Client) *Builder {
	b.client = cli
	return b
}

// LogChannel injects a channel receiving the log
// messages that will end up going to Loki in Stream().
func (b *Builder) LogChannel(l chan string) *Builder {
	b.logChannel = l
	return b
}

// ErrChannel injects a channel aggregating errors
// from Stream().
func (b *Builder) ErrChannel(e chan error) *Builder {
	b.errChannel = e
	return b
}

type loki struct {
	url        string
	client     *http.Client
	logChannel chan string
	errChannel chan error
}

type lokiDTO struct {
	streams []streams `json:"streams"`
}

type streams struct {
	stream map[string]string `json:"stream"`
	values [][]string        `json:"values"`
}

// Stream does a POST request of the logChannel
// into the Loki API.
func (l *loki) Stream() {
	for log := range l.logChannel {
		msg := &lokiDTO{
			streams: []streams{
				{
					Stream: "",
					Values: [][]string{
						{""},
					},
				},
			},
		}

		body, err := json.Marshal(msg)
		if err != nil {
			l.errChannel <- err
		}

		req, err := http.NewRequest("POST", l.url, bytes.NewBuffer(body))
		if err != nil {
			l.errChannel <- err
		}
		req.Header.Add("Content-Type", "application/json")

		res, err := l.client.Do(req)
		if err != nil {
			l.errChannel <- err
		}

		if res.StatusCode != 204 {
			buf := new(bytes.Buffer)
			buf.ReadFrom(res.Body)
			l.errChannel <- fmt.Errorf("%s - %s", res.Status, buf.String())
		}
	}
}

// Close will close the injected logChannel.
// Unprocessed items will still get streamed.
func (l *loki) Close() {
	close(l.logChannel)
}
