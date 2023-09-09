package logstream

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/phil-inc/admiral/pkg/backend"
	"github.com/phil-inc/admiral/pkg/state"
	"github.com/phil-inc/admiral/pkg/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type logstream struct {
	rawLogChannel chan backend.RawLog
	state         *state.SharedMutable
	pod           *v1.Pod
	container     v1.Container
	reader        *bufio.Reader
	stream        io.ReadCloser
}

type builder struct {
	rawLogChannel chan backend.RawLog
	state         *state.SharedMutable
	pod           *v1.Pod
	container     v1.Container
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
		state:         b.state,
		pod:           b.pod,
		container:     b.container,
	}
}

func (l *logstream) Stream() {
	l.Open(nil)

	l.Read()

	for {
		name := utils.GenerateUniqueContainerName(l.pod, l.container)
		if l.state.Get(name) == state.RUNNING {
			t := metav1.NewTime(time.Now())
			l.Open(t.DeepCopy())
			l.Read()
		} else {
			break
		}
	}
}

func (l *logstream) Open(since *metav1.Time) {
	var err error
	ctx := context.Background()

	if l.state.GetKubeClient() == nil {
		l.state.Error(errors.New("missing kube client"))
		return
	}

	l.stream, err = l.state.GetKubeClient().CoreV1().Pods(l.pod.Namespace).GetLogs(l.pod.Name,
		&v1.PodLogOptions{
			Container:  l.container.Name,
			Follow:     true,
			Timestamps: false,
			SinceTime:  since,
		}).Stream(ctx)

	if err != nil {
		if err != io.EOF {
			l.state.Error(err)
		}
		return
	}

	l.reader = bufio.NewReader(l.stream)
}

func (l *logstream) Read() {
	for {
		line, err := l.reader.ReadString('\n')

		if err != nil {
			return
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

		timestamp, err := l.getTimestamp(msg)
		if err != nil {
			l.state.Error(err)
		}

		raw := backend.RawLog{
			Log:       msg,
			Metadata:  formatLogMetadata(metadata),
			Timestamp: timestamp,
		}

		l.rawLogChannel <- raw
	}
}

func (l *logstream) getTimestamp(msg string) (string, error) {
	timeKey := l.pod.Annotations["admiral.io/time-key"]

	// try parsing what could be json
	if len(msg) > 0 && msg[0:1] == "{" {
		var log map[string]interface{}

		err := json.Unmarshal([]byte(msg), &log)
		if err != nil {
			return fmt.Sprintf("%d", time.Now().UnixNano()), err
		}

		if v, ok := log[timeKey]; ok {
			t, err := time.Parse(time.RFC3339, fmt.Sprintf("%v", v))
			if err != nil {
				return fmt.Sprintf("%d", time.Now().UnixNano()), err
			}
			return fmt.Sprintf("%d", t.UnixNano()), nil
		}
	}

	return fmt.Sprintf("%d", time.Now().UnixNano()), nil
}

func formatLogMetadata(m map[string]string) map[string]string {
	lm := make(map[string]string)
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
	return lm
}
