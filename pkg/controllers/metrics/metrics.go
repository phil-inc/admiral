package metrics

import (
	"fmt"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/metrics_handlers"
	"github.com/phil-inc/admiral/pkg/utils"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type MetricsController struct {
	informerFactory informers.SharedInformerFactory
	podInformer     coreinformers.PodInformer
	config          *config.Config
	metricstreams   map[string]*metrics_handlers.Metricstream
	handler         metrics_handlers.MetricsHandler
	client          kubernetes.Interface
	metricsCh       chan (metrics_handlers.MetricBatch)
}

// Instantiates a controller for watching and handling metrics
func NewMetricsController(informerFactory informers.SharedInformerFactory, metricsHandler metrics_handlers.MetricsHandler, config *config.Config) *MetricsController {
	podInformer := informerFactory.Core().V1().Pods()

	c := &MetricsController{
		informerFactory: informerFactory,
		podInformer:     podInformer,
		config:          config,
		metricstreams:   make(map[string]*metrics_handlers.Metricstream),
		handler:         metricsHandler,
		metricsCh:       make(chan metrics_handlers.MetricBatch),
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

// Watch creates the informerFactory and initializes the pod watcher
func (c *MetricsController) Watch() chan struct{} {
	metricsStop := make(chan struct{})
	err := c.Run(metricsStop)
	if err != nil {
		logrus.Fatal(err)
	}

	return metricsStop
}

func (c *MetricsController) Run(stopCh chan struct{}) error {
	c.informerFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.podInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync")
	}

	go c.handler.Handle(c.metricsCh)
	return nil
}

func (c *MetricsController) onPodAdd(obj interface{}) {
	pod := obj.(*api_v1.Pod)

	if c.podIsInConfig(pod) {
		if pod.Status.Phase == api_v1.PodRunning {
			c.newPod(pod)
		}
	}
}

func (c *MetricsController) onPodUpdate(old, new interface{}) {
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

func (c *MetricsController) onPodDelete(obj interface{}) {
	pod := obj.(*api_v1.Pod)
	if c.podIsInConfig(pod) {
		c.deletedPod(pod)
	}
}

func (c *MetricsController) newPod(pod *api_v1.Pod) {
	stream := metrics_handlers.NewMetricStream(pod, c.handler)
	_, exists := c.metricstreams[pod.Name]

	if exists {
		if !c.metricstreams[pod.Name].Finished {
			return
		}
	}

	if !exists {
		c.metricstreams[pod.Name] = stream
	}

	r, err := utils.GetRestConfig()
	if err != nil {
		logrus.Fatalf("Cannot find REST config: %s", err)
	}
	stream.Start(r, c.metricsCh)
}

func (c *MetricsController) finishedPod(pod *api_v1.Pod) {
	if c.metricstreams[pod.Name] != nil && !c.metricstreams[pod.Name].Finished {
		c.metricstreams[pod.Name].Finish()
	}
}

func (c *MetricsController) deletedPod(pod *api_v1.Pod) {
	if c.metricstreams[pod.Name] != nil {
		c.metricstreams[pod.Name].Delete()
		delete(c.metricstreams, pod.Name)
	}
}

func (c *MetricsController) podIsInConfig(pod *api_v1.Pod) bool {
	// If it is in the apps array, return true
	for _, v := range c.config.Metrics.Apps {
		if pod.ObjectMeta.Labels["app"] == v {
			return true
		}
	}
	return false
}
