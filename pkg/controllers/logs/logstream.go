package logs

import (
	"bufio"
	"context"
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
	logrus.Printf("Starting logstream %s.%s.%s", l.namespace, l.pod, l.container)
	l.closed = make(chan struct{})

	go func() {
		stream, err := l.clientset.CoreV1().Pods(l.namespace).GetLogs(l.pod, &api_v1.PodLogOptions{
			Container:  l.container,
			Follow:     true,
			Timestamps: true,
			SinceTime:  t,
		}).Stream(context.Background())
		if err != nil {
			logrus.Errorf("Failed opening logstream %s.%s.%s: %s", l.namespace, l.pod, l.container, err)
		}

		if stream != nil {
			logrus.Printf("Started logstream %s.%s.%s", l.namespace, l.pod, l.container)

			go func() {
				<-l.closed
				logrus.Printf("Received closure for logstream %s.%s.%s", l.namespace, l.pod, l.container)
				stream.Close()
			}()

			logs := bufio.NewScanner(stream)

			err := l.Scan(logs)
			if err != nil {
				logrus.Errorf("Error scanning logs %s: %s", l.pod, err)
			}
			logrus.Printf("Reached end of logstream scope: %s", l.pod)
		}
	}()
}

func (l *logstream) Scan(logs *bufio.Scanner) error {
	for {
		if l.Finished {
			return nil
		}
		if logs.Err() != nil {
			return logs.Err()
		}
		if logs.Scan() {
			logMetaData := make(map[string]string)
			for k, v := range l.podLabels {
				logMetaData[k] = v
			}
			logMetaData["pod"] = l.pod
			logMetaData["namespace"] = l.namespace

			err := l.logstore.Stream(logs.Text(), formatLogMetadata(logMetaData))
			if err != nil {
				return err
			}
		} else {
			logrus.Printf("Empty log scanner: %s", l.pod)
			logrus.Printf("Waiting one minute then restarting %s", l.pod)
			t := metav1.NewTime(time.Now())
			time.Sleep(1 * time.Minute)
			if !l.Finished {
				l.Restart(t.DeepCopy())
				return nil
			}
		}
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
	l.Start(t)
}
