package logs

import (
	"bufio"
	"context"
	"io"
	"strings"
	"time"

	"github.com/phil-inc/admiral/pkg/logstores"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type logstream struct {
	Finished  bool
	closed    chan struct{}
	namespace string
	pod       string
	container string
	podLabels map[string]string
	logstore  logstores.Logstore
	clientset kubernetes.Interface
}

type logEntry struct {
	text     string
	metadata map[string]string
	err      error
}

func NewLogstream(namespace string, pod string, container string, podLabels map[string]string, logstore logstores.Logstore, clientset kubernetes.Interface) *logstream {
	return &logstream{
		Finished:  false,
		namespace: namespace,
		pod:       pod,
		container: container,
		podLabels: podLabels,
		logstore:  logstore,
		clientset: clientset,
	}
}

func (l *logstream) Start(t *metav1.Time) {
	l.closed = make(chan struct{})

	stream, err := l.clientset.CoreV1().Pods(l.namespace).GetLogs(l.pod, &api_v1.PodLogOptions{
		Container:  l.container,
		Follow:     true,
		Timestamps: true,
		SinceTime:  t,
	}).Stream(context.Background())
	if err == nil {
		defer stream.Close()
	}

	entry := make(chan logEntry)

	go func() {
		if err != nil {
			logrus.Errorf("Failed opening logstream %s.%s.%s: %s", l.namespace, l.pod, l.container, err)
		} else {
			l.Scan(stream, entry)
		}
		close(entry)
	}()

	done := make(chan error)

	go func() {
		for result := range entry {
			if result.err != nil {
				done <- result.err
				return
			} else {
				err := l.logstore.Stream(result.text, result.metadata)
				if err != nil {
					done <- err
					break
				}
			}
		}
	}()

	select {
	case err := <-done:
		logrus.Errorf("%s\t%s\t%s\t%s", l.namespace, l.pod, l.container, err)
	case <-context.Background().Done():
		logrus.Printf("DONE: %s\t%s\t%s", l.namespace, l.pod, l.container)
	}
}

func (l *logstream) Scan(stream io.ReadCloser, ch chan logEntry) {
	bufReader := bufio.NewReader(stream)
	eof := false

	for !eof {
		line, err := bufReader.ReadString('\n')
		if err == io.EOF {
			eof = true
			if line == "" {
				break
			}
		} else if err != nil && err != io.EOF {
			ch <- logEntry{err: err}
			break
		}

		line = strings.TrimSpace(line)
		md := make(map[string]string)
		for k, v := range l.podLabels {
			md[k] = v
		}
		md["pod"] = l.pod
		md["namespace"] = l.namespace

		ch <- logEntry{text: line, metadata: formatLogMetadata(md)}
	}
}

func (l *logstream) Finish() {
	logrus.Printf("Logstream finished: %s.%s.%s", l.namespace, l.pod, l.container)
	l.Finished = true
}

func (l *logstream) Delete() {
	l.Finish()
	logrus.Printf("Logstream deleted: %s.%s.%s", l.namespace, l.pod, l.container)
	close(l.closed)
}

func (l *logstream) Restart(t *metav1.Time) {
	logrus.Printf("Logstream restarted at %s: %s.%s.%s", t, l.namespace, l.pod, l.container)
	close(l.closed)
	time.Sleep(1 * time.Second)
	l.Start(t)
}
