package logs

import (
	"bufio"
	"context"
	"time"

	"github.com/phil-inc/admiral/pkg/logstores"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
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
}

func NewLogstream(namespace string, pod string, container string, podLabels map[string]string, logstore logstores.Logstore) *logstream {
	return &logstream{
		Finished:  false,
		closed:    make(chan struct{}),
		namespace: namespace,
		pod:       pod,
		container: container,
		podLabels: podLabels,
		logstore:  logstore,
	}
}

func (l *logstream) Start(clientset kubernetes.Interface) {
	logrus.Printf("Starting logstream %s.%s.%s", l.namespace, l.pod, l.container)

	go func() {
		stream, err := clientset.CoreV1().Pods(l.namespace).GetLogs(l.pod, &api_v1.PodLogOptions{
			Container:  l.container,
			Follow:     true,
			Timestamps: true,
		}).Stream(context.Background())
		if err != nil {
			logrus.Errorf("Failed opening logstream %s.%s.%s: %s", l.namespace, l.pod, l.container, err)
		}

		if stream != nil {
			defer stream.Close()

			logrus.Printf("Started logstream %s.%s.%s", l.namespace, l.pod, l.container)

			go func() {
				<-l.closed
				logrus.Printf("Received closure for logstream %s.%s.%s", l.namespace, l.pod, l.container)
				l.Finish()
				stream.Close()
			}()

			logs := bufio.NewScanner(stream)

			err := l.Scan(logs)
			if err != nil {
				logrus.Errorf("Error scanning logs: %s", err)
			}
		}
	}()
}

func (l *logstream) Scan(logs *bufio.Scanner) error {
	for {
		logs.Scan()
		time.Sleep(1 * time.Second)
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
		if logs.Err() != nil {
			return logs.Err() 
		}
	}
}

func (l *logstream) Finish() {
	logrus.Printf("Logstream finished: %s.%s.%s", l.namespace, l.pod, l.container)
	l.Finished = true
}

func (l *logstream) Delete() {
	logrus.Printf("Logstream deleted: %s.%s.%s", l.namespace, l.pod, l.container)
	close(l.closed)
}
