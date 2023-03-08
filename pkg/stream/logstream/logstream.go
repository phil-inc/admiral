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

type logstream struct {
	rawLogChannel chan backend.RawLog
	state *state.SharedMutable
	pod *v1.Pod
	container v1.Container
	reader *bufio.Reader
	stream io.ReadCloser
}

type builder struct {
	rawLogChannel chan backend.RawLog
	state *state.SharedMutable
	pod *v1.Pod
	container v1.Container
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

func (b *builder) Build() *logstream {
	return &logstream{
		rawLogChannel: b.rawLogChannel,
		state: b.state,
		pod: b.pod,
		container: b.container,
	}
}

func (l *logstream) Stream() {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 300)
	defer cancel()

	l.stream, err = l.state.GetKubeClient().CoreV1().Pods(l.pod.Namespace).GetLogs(l.pod.Name,
						&v1.PodLogOptions{
							Container: l.container.Name,
							Follow: true,
							Timestamps: false,
						}).Stream(ctx)

	if err != nil {
		l.state.Error(err)
	}

	l.reader = bufio.NewReader(l.stream)
}

func (l *logstream) Read() {
	for {
		line, err := l.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			l.state.Error(err)
		}

		if line == "" {
			continue
		}

		msg := strings.TrimSpace(line)

		metadata := make(map[string]string)

		if l.pod.Labels != nil {
			metadata = l.pod.Labels
		}

		metadata["pod"] = l.pod.Name
		metadata["namespace"] = l.pod.Namespace

		raw := backend.RawLog{
			Log: msg,
			Metadata: metadata,
		}

		l.rawLogChannel <- raw

		// discard buffered data 
	//	l.reader.Discard(len(line))
	} 
	l.stream.Close()
}
