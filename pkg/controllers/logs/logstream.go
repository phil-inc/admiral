package logs

import (
	"bufio"
	"context"
	"io"
	"strings"
	"time"

	"github.com/phil-inc/admiral/pkg/logstores"
	"github.com/phil-inc/admiral/pkg/utils"
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
	logCh     chan utils.LogEntry
}

func NewLogstream(namespace string, pod string, container string, podLabels map[string]string, logstore logstores.Logstore, clientset kubernetes.Interface, logCh chan utils.LogEntry) *logstream {
	return &logstream{
		Finished:  false,
		namespace: namespace,
		pod:       pod,
		container: container,
		podLabels: podLabels,
		logstore:  logstore,
		clientset: clientset,
		logCh:     logCh,
	}
}

func (l *logstream) Start(t *metav1.Time) {
	l.closed = make(chan struct{})
	restart := make(chan error)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	stream, err := l.clientset.CoreV1().Pods(l.namespace).GetLogs(l.pod, &api_v1.PodLogOptions{
		Container:  l.container,
		Follow:     true,
		Timestamps: true,
		SinceTime:  t,
	}).Stream(ctx)
	if err != nil {
		restart <- err
	}

	defer stream.Close()

	go l.Scan(stream, l.logCh, restart)

	select {
	case <-ctx.Done():
		if ctx.Err() != nil {
			restart <- ctx.Err()
		}
	case err := <-restart:
		logrus.Errorf("%s\t%s\t%s\t%s", l.namespace, l.pod, l.container, err)
		t := metav1.NewTime(time.Now())
		time.Sleep(60 * time.Second)
		if !l.Finished {
			l.Flush(t.DeepCopy())
		}
	}

	<-l.closed
	logrus.Printf("DONE: %s\t%s\t%s", l.namespace, l.pod, l.container)
}

func (l *logstream) Scan(stream io.ReadCloser, ch chan utils.LogEntry, restart chan error) {
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
			restart <- err
			break
		}

		line = strings.TrimSpace(line)
		md := make(map[string]string)
		for k, v := range l.podLabels {
			md[k] = v
		}
		md["pod"] = l.pod
		md["namespace"] = l.namespace

		ch <- utils.LogEntry{Text: line, Metadata: formatLogMetadata(md)}
	}
}

func (l *logstream) Finish() {
	logrus.Printf("Logstream finished: %s\t%s\t%s", l.namespace, l.pod, l.container)
	l.Finished = true
}

func (l *logstream) Delete() {
	l.Finish()
	logrus.Printf("Logstream deleted: %s\t%s\t%s", l.namespace, l.pod, l.container)
	close(l.closed)
}

func (l *logstream) Flush(t *metav1.Time) {
	close(l.closed)
	time.Sleep(1 * time.Second)
	go l.Start(t)
	logrus.Printf("Flushing %s\t %s\t %s\t %s", t, l.namespace, l.pod, l.container)
}
