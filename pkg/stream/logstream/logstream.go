package logstream

import (
	"bufio"
	"context"
	"io"
	"strings"

	"github.com/phil-inc/admiral/pkg/backend"
	"github.com/phil-inc/admiral/pkg/state"
	v1 "k8s.io/api/core/v1"
)

type stream struct {
	rawLogChannel chan backend.RawLog
	errChannel chan error
	state *state.SharedMutable
	pod *v1.Pod
	container v1.Container
	reader io.ReadCloser
}

type builder struct {
	rawLogChannel chan backend.RawLog
	errChannel chan error
	state *state.SharedMutable
	pod *v1.Pod
	container v1.Container
	reader io.ReadCloser
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

func (b *builder) Container(container v1.Container) *builder {
	b.container = container
	return b
}

func (b *builder) Pod(pod *v1.Pod) *builder {
	b.pod = pod
	return b
}

func (b *builder) ErrChannel(errChannel chan error) *builder {
	b.errChannel = errChannel
	return b
}

func (b *builder) Build() *stream {
	return &stream{
		rawLogChannel: b.rawLogChannel,
		errChannel: b.errChannel,
		state: b.state,
		pod: b.pod,
		container: b.container,
	}
}

func (s *stream) Stream() {
	ctx, cancel := context.WithTimeout(context.Background(), 300)
	defer cancel()

	stream, err := s.state.GetKubeClient().CoreV1().Pods(s.pod.Namespace).GetLogs(s.pod.Name,
						&v1.PodLogOptions{
							Container: s.container.Name,
							Follow: true,
							Timestamps: false,
						}).Stream(ctx)

	if err != nil {
		s.state.Error(err)
	}

	defer stream.Close()

	reader := bufio.NewReader(stream)

	for {
		b, err := reader.Peek(1)
		if err != nil {
			if err == io.EOF {
				break
			}
			s.state.Error(err)
		}

		// if the next byte is not a newline, continue
		if len(b) == 0 || b[0] != '\n' {
			_, err := reader.ReadBytes('\n')
			if err != nil {
				s.state.Error(err)
			}
			continue
		}

		// if the next byte is a newline, read the line
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			s.state.Error(err)
		}

		msg := strings.TrimSpace(line)
		metadata := s.pod.Labels
		metadata["pod"] = s.pod.Name
		metadata["namespace"] = s.pod.Namespace

		raw := backend.RawLog{
			Log: msg,
			Metadata: metadata,
		}

		s.rawLogChannel <- raw

		// discard buffered data 
		reader.Discard(len(line))
	}
}
