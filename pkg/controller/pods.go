package controller

import (
	"sync"
	"bufio"
	"fmt"
	"context"
	"strings"
	"github.com/sirupsen/logrus"
	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/logstores"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/kubernetes"
)

type PodController struct {
	informerFactory informers.SharedInformerFactory
	podInformer     coreinformers.PodInformer
	clientset       kubernetes.Interface
	config          *config.Config
	logstream       map[string]chan struct{}
	logstreamMu     sync.Mutex
	logstore        logstores.Logstore
}

// Instantiates a controller for watching and handling pods
func NewPodController(informerFactory informers.SharedInformerFactory, clientset kubernetes.Interface, config *config.Config, logstore logstores.Logstore) *PodController {
	podInformer := informerFactory.Core().V1().Pods()

	c := &PodController{
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

func (c *PodController) Run(stopCh chan struct{}) error {
	c.informerFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.podInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync")
	}
	return nil
}

func (c *PodController) onPodAdd(obj interface{}) {
	pod := obj.(*api_v1.Pod)
	logrus.Printf("Pod added: %s", pod.ObjectMeta.Name)
	if c.podIsInConfig(pod) {
		if pod.Status.Phase == api_v1.PodRunning {
			logrus.Printf("Streaming logs from %s", pod.ObjectMeta.Name)
			c.streamLogsFromPod(pod)
		}
	}
}

func (c *PodController) onPodUpdate(old, new interface{}) {
	oldPod := old.(*api_v1.Pod)
	newPod := new.(*api_v1.Pod)

	if c.podIsInConfig(newPod) {
		// Pod is running & was not previously
		if newPod.Status.Phase == api_v1.PodRunning && oldPod.Status.Phase != api_v1.PodRunning {
			logrus.Printf("Streaming logs from %s", newPod.ObjectMeta.Name)
			c.streamLogsFromPod(newPod)
		}

		// Pod is not running, but was
		if newPod.Status.Phase != api_v1.PodRunning && oldPod.Status.Phase == api_v1.PodRunning {
			logrus.Printf("Stopped streaming logs from %s", newPod.ObjectMeta.Name)
			c.stopLogStreamFromPod(newPod)
		}
	}
}

func (c *PodController) onPodDelete(obj interface{}) {
	pod := obj.(*api_v1.Pod)

	logrus.Printf("Pod deleted: %s", pod.ObjectMeta.Name)
	if c.podIsInConfig(pod) {
		logrus.Printf("Stopped streaming logs from %s", pod.ObjectMeta.Name)
		c.stopLogStreamFromPod(pod)
	}
}

func (c *PodController) streamLogsFromPod(pod *api_v1.Pod) {
	// add all of the containers in the pod to the logstream
	// stream the logs
	for _, container := range pod.Spec.Containers {
		con := container
		// process each log stream concurrently
		go func() {
			name := getLogstreamName(pod, con)
			logrus.Printf("Opening stream from %s", name)

			c.logstreamMu.Lock()
			c.logstream[name] = make(chan struct{})
			c.logstreamMu.Unlock()

			sinceSeconds := int64(1)

			stream, err := c.clientset.CoreV1().Pods(pod.ObjectMeta.Namespace).GetLogs(pod.ObjectMeta.Name, &api_v1.PodLogOptions{
				Container: con.Name,
				Follow: true,
				Timestamps: true,
				SinceSeconds: &sinceSeconds,
			}).Stream(context.Background())
			if err != nil {
				logrus.Error(err)
			}
			defer stream.Close()

			// concurrently wait for the receiver to close, then close the stream
			go func() {
				c.logstreamMu.Lock()
				<-c.logstream[name]
				c.logstreamMu.Unlock()
				stream.Close()
			}()

			logs := bufio.NewScanner(stream)

			for logs.Scan() {
				// do something with each log line
				err := c.logstore.Stream(logs.Text(), formatLogMetadata(pod.ObjectMeta.Labels))
				if err != nil {
					logrus.Fatalf("Failed streaming log to logstore: %s", err)
				}
			}
		}()
	}
}

func (c *PodController) stopLogStreamFromPod(pod *api_v1.Pod) {
	for _, container := range pod.Spec.Containers {
		name := getLogstreamName(pod, container)
		c.logstreamMu.Lock()
		close(c.logstream[name])
		c.logstreamMu.Unlock()
	}
}

func (c *PodController) podIsInConfig(pod *api_v1.Pod) bool {
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
		parsedK = strings.ReplaceAll(parsedK, "\\", "-")
		parsedK = strings.ReplaceAll(parsedK, "/", "-")
		parsedV := strings.ReplaceAll(v, "\\", "-")
		parsedV = strings.ReplaceAll(parsedV, ".", "_")
		parsedV = strings.ReplaceAll(parsedV, "/", "-")
		l[parsedK] = parsedV
	}
	return l
}