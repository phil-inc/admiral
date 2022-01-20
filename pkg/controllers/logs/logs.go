package logs

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/logstores"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type LogController struct {
	informerFactory informers.SharedInformerFactory
	podInformer     coreinformers.PodInformer
	clientset       kubernetes.Interface
	config          *config.Config
	logstream       map[string]chan struct{}
	logstore        logstores.Logstore
}

// Instantiates a controller for watching and handling logs
func NewLogController(informerFactory informers.SharedInformerFactory, clientset kubernetes.Interface, config *config.Config, logstore logstores.Logstore) *LogController {
	podInformer := informerFactory.Core().V1().Pods()

	c := &LogController{
		informerFactory: informerFactory,
		podInformer:     podInformer,
		clientset:       clientset,
		config:          config,
		logstream:       make(map[string]chan struct{}),
		logstore:        logstore,
	}

	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onPodAdd,
			UpdateFunc: c.onPodUpdate,
			DeleteFunc: c.onPodDelete,
		},
	)

	return c
}

// Watch creates the informerFactory and initializes the log watcher
func (c *LogController) Watch() chan struct{} {
	logStop := make(chan struct{})
	err := c.Run(logStop)
	if err != nil {
		logrus.Fatal(err)
	}
	return logStop
}

func (c *LogController) Run(stopCh chan struct{}) error {
	c.informerFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.podInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync")
	}
	return nil
}

func (c *LogController) onPodAdd(obj interface{}) {
	pod := obj.(*api_v1.Pod)

	if c.podIsInConfig(pod) {
		if pod.Status.Phase == api_v1.PodRunning {
			c.streamLogsFromPod(pod)
		}
	}
}

func (c *LogController) onPodUpdate(old, new interface{}) {
	pod := new.(*api_v1.Pod)

	if c.podIsInConfig(pod) {
		switch pod.Status.Phase {
		case api_v1.PodRunning:
			c.streamLogsFromPod(pod)
		case api_v1.PodSucceeded, api_v1.PodFailed:
			c.stopLogStreamFromPod(pod)
		}
	}
}

func (c *LogController) onPodDelete(obj interface{}) {
	pod := obj.(*api_v1.Pod)
	if c.podIsInConfig(pod) {
		c.stopLogStreamFromPod(pod)
	}
}

func (c *LogController) streamLogsFromPod(pod *api_v1.Pod) {
	// add all of the containers in the log to the logstream
	// stream the logs
	for _, container := range pod.Spec.Containers {
		con := container
		name := getLogstreamName(pod, con)

		// if the entry already exists in the logstream, skip
		if _, ok := c.logstream[name]; ok {
			continue
		}

		logrus.Printf("Opening stream from %s", name)
		c.logstream[name] = make(chan struct{})

		// process each log stream concurrently
		go func() {
			stream, err := c.clientset.CoreV1().Pods(pod.ObjectMeta.Namespace).GetLogs(pod.ObjectMeta.Name, &api_v1.PodLogOptions{
				Container:  con.Name,
				Follow:     true,
				Timestamps: true,
			}).Stream(context.Background())

			if err != nil {
				logrus.Errorf("Failed opening logstream %s: %s", name, err)
			} else {
				logrus.Printf("Opened logstream: %s", name)
				defer close(c.logstream[name])

				// concurrently wait for the receiver to close, then close the stream
				go func() {
					<-c.logstream[name]
					stream.Close()
					delete(c.logstream, name)
					logrus.Printf("Received logstream closure: %s", name)
				}()

				logs := bufio.NewScanner(stream)

				for logs.Scan() {
					// do something with each log line

					// prepare log meta data
					logMetaData := make(map[string]string)
					for k, v := range pod.ObjectMeta.Labels {
						logMetaData[k] = v
					}
					logMetaData["pod"] = pod.GetName()
					logMetaData["namespace"] = pod.GetNamespace()
					logMetaData["cluster"] = pod.GetClusterName()

					err := c.logstore.Stream(logs.Text(), formatLogMetadata(logMetaData))
					if err != nil {
						logrus.Errorf("Failed streaming log to logstore: %s", err)
					}
				}

				if logs.Err() != nil {
					logrus.Errorf("Scanner failed %s: %s", name, logs.Err())
				}

				logrus.Printf("Scanner for %s closed", name)
			}
		}()
	}
}

func (c *LogController) stopLogStreamFromPod(pod *api_v1.Pod) {
	logrus.Printf("Stopped streaming logs from %s", pod.ObjectMeta.Name)
	for _, container := range pod.Spec.Containers {
		name := getLogstreamName(pod, container)
		if stream, ok := c.logstream[name]; ok {
			close(stream)
		}
	}
}

func (c *LogController) podIsInConfig(pod *api_v1.Pod) bool {
	// If it is in the apps array, return true
	for _, v := range c.config.Logstream.Apps {
		if pod.ObjectMeta.Labels["app"] == v {
			return true
		}
	}
	return false
}

func getLogstreamName(pod *api_v1.Pod, container api_v1.Container) string {
	name := fmt.Sprintf("%s.%s.%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, container.Name)
	return name
}

func formatLogMetadata(m map[string]string) map[string]string {
	l := make(map[string]string)
	for k, v := range m {
		parsedK := strings.ReplaceAll(k, ".", "_")
		parsedK = strings.ReplaceAll(parsedK, "\\", "_")
		parsedK = strings.ReplaceAll(parsedK, "-", "_")
		parsedK = strings.ReplaceAll(parsedK, "/", "_")
		parsedV := strings.ReplaceAll(v, "\\", "_")
		parsedV = strings.ReplaceAll(parsedV, "-", "_")
		parsedV = strings.ReplaceAll(parsedV, ".", "_")
		parsedV = strings.ReplaceAll(parsedV, "/", "_")
		l[parsedK] = parsedV
	}
	return l
}
