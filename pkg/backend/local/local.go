package local

import (
	"fmt"

	"github.com/phil-inc/admiral/pkg/backend"
)

type Builder struct {
	logChannel   chan backend.RawLog
	eventChannel chan string
	errChannel   chan error
}

func New() *Builder {
	return &Builder{}
}

func (b *Builder) Build() *local {
	return &local{
		logChannel:   b.logChannel,
		eventChannel: b.eventChannel,
		errChannel:   b.errChannel,
	}
}

func (b *Builder) LogChannel(l chan backend.RawLog) *Builder {
	b.logChannel = l
	return b
}

func (b *Builder) ErrChannel(e chan error) *Builder {
	b.errChannel = e
	return b
}

func (b *Builder) EventChannel(e chan string) *Builder {
	b.eventChannel = e
	return b
}

type local struct {
	logChannel   chan backend.RawLog
	errChannel   chan error
	eventChannel chan string
}

func (l *local) Stream() {
	go l.streamLogs()
	go l.streamEvents()
}

func (l *local) streamLogs() {
	for raw := range l.logChannel {
		// fmt.Println(raw.Log)
		fmt.Println(raw.Metadata)
	}
}

func (l *local) streamEvents() {
	for event := range l.eventChannel {
		fmt.Println(event)
	}
}

func (l *local) Close() {
	close(l.logChannel)
}
