package logstream

import (
	"github.com/phil-inc/admiral/pkg/backend"
	"github.com/phil-inc/admiral/pkg/state"
)

type stream struct {
	rawLogChannel chan backend.RawLog
	state *state.SharedMutable
}

type builder struct {
	rawLogChannel chan backend.RawLog
	state *state.SharedMutable
}

func New() *builder {
	return &builder{}
}

func (b *builder) RawLogChannel(rawLogChannel chan backend.RawLog) *builder {
	b.rawLogChannel = rawLogChannel
	return b
}

func (b *builder) State(state *state.SharedMutable) *builder {
	b.state = state
	return b
}

func (b *builder) Build() *stream {
	return &stream{
		rawLogChannel: b.rawLogChannel,
		state: b.state,
	}
}

func (s *stream) Start() {}
