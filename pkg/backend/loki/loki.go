package loki

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/phil-inc/admiral/pkg/backend"
	"github.com/phil-inc/admiral/pkg/utils"
)

type Builder struct {
	url        string
	client     *http.Client
	logChannel chan backend.RawLog
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
func (b *Builder) LogChannel(l chan backend.RawLog) *Builder {
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
	logChannel chan backend.RawLog
	errChannel chan error
	open       chan bool
	mutex      sync.RWMutex
}

type lokiDTO struct {
	Streams []streams `json:"streams"`
}

type streams struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// Stream does a POST request of the logChannel
// into the Loki API.
func (l *loki) Stream() {
	for raw := range l.logChannel {
		dto := l.rawLogToDTO(raw)

		err := utils.Send(dto, "POST", l.url, l.client)
		if err != nil {
			l.errChannel <- err
		}
	}
}

func (l *loki) rawLogToDTO(r backend.RawLog) *lokiDTO {
	return &lokiDTO{
		Streams: []streams{
			{
				Stream: l.formatLogMetadata(r.Metadata),
				Values: [][]string{{r.Timestamp, r.Log}},
			},
		},
	}
}

func (l *loki) formatLogMetadata(m map[string]string) map[string]string {
	lm := make(map[string]string)
	l.mutex.Lock()
	for k, v := range m {
		parsedK := strings.ReplaceAll(k, ".", "_")
		parsedK = strings.ReplaceAll(parsedK, "\\", "_")
		parsedK = strings.ReplaceAll(parsedK, "-", "_")
		parsedK = strings.ReplaceAll(parsedK, "/", "_")
		parsedV := strings.ReplaceAll(v, "\\", "_")
		parsedV = strings.ReplaceAll(parsedV, "-", "_")
		parsedV = strings.ReplaceAll(parsedV, ".", "_")
		parsedV = strings.ReplaceAll(parsedV, "/", "_")
		lm[parsedK] = parsedV
	}
	l.mutex.Unlock()
	return lm
}

// Close will close the injected logChannel.
// Unprocessed items will still get streamed.
func (l *loki) Close() {
	close(l.logChannel)
}
