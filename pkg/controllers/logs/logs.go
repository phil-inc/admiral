package logs

import (
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
	logstreams      map[string]*logstream
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
		logstreams:      make(map[string]*logstream),
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
			c.newPod(pod)
		}
	}
}

func (c *LogController) onPodUpdate(old, new interface{}) {
	pod := new.(*api_v1.Pod)

	if c.podIsInConfig(pod) {
		switch pod.Status.Phase {
		case api_v1.PodRunning:
			c.newPod(pod)
		case api_v1.PodSucceeded, api_v1.PodFailed:
			c.finishedPod(pod)
		}
	}
}

func (c *LogController) onPodDelete(obj interface{}) {
	pod := obj.(*api_v1.Pod)
	if c.podIsInConfig(pod) {
		c.deletedPod(pod)
	}
}

func (c *LogController) newPod(pod *api_v1.Pod) {
	for _, container := range pod.Spec.Containers {

		if !ignoreContainer(pod, container.Name, c.config.IgnoreContainers) {

			name := getLogstreamName(pod, container)
			stream := NewLogstream(pod.Namespace, pod.Name, container.Name, pod.Labels, c.logstore)
			_, exists := c.logstreams[name]

			if exists {
				if !c.logstreams[name].Finished {
					continue
				}
			}

			if !exists {
				c.logstreams[name] = stream
			}

			stream.Start(c.clientset)
		}
	}
}

func (c *LogController) finishedPod(pod *api_v1.Pod) {
	for _, container := range pod.Spec.Containers {
		name := getLogstreamName(pod, container)

		if c.logstreams[name] == nil {
			continue
		}

		if c.logstreams[name].Finished {
			continue
		}

		c.logstreams[name].Finish()
	}
}

func (c *LogController) deletedPod(pod *api_v1.Pod) {
	for _, container := range pod.Spec.Containers {
		name := getLogstreamName(pod, container)

		if c.logstreams[name] == nil {
			continue
		}

		c.logstreams[name].Delete()
		delete(c.logstreams, name)
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

func ignoreContainer(pod *api_v1.Pod, containerName string, ignoreContainers []string) bool {
	for _, c := range ignoreContainers {

		if c == containerName {
			return true
		}
	}

	labels := strings.Split(pod.ObjectMeta.Labels["ignore_logs"], ",")
	for _, label := range labels {

		if label == containerName {
			return true
		}
	}
	// if neither the pod labels nor the admiral config values match the container name in the running pod
	// then continue to process logs for that container
	return false
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
