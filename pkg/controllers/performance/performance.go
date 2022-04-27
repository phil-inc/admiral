package performance

import (
	"fmt"
	"net/http"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/target"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type PerformanceController struct {
	informerFactory informers.SharedInformerFactory
	podInformer     coreinformers.PodInformer
	clientset       kubernetes.Interface
	config          *config.Config
	target          target.Target
	httpClient      http.Client
}

// Instantiates a controller for watching and handling performance testing
func NewPerformanceController(informerFactory informers.SharedInformerFactory, clientset kubernetes.Interface, config *config.Config, target target.Target) *PerformanceController {
	podInformer := informerFactory.Core().V1().Pods()
	httpClient := http.Client{}

	ctrl := &PerformanceController{
		informerFactory: informerFactory,
		podInformer:     podInformer,
		clientset:       clientset,
		config:          config,
		target:          target,
		httpClient:      httpClient,
	}

	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.onPodAdd,
			UpdateFunc: ctrl.onPodUpdate,
		},
	)

	return ctrl
}

// Watch creates the informerFactory and initializes the log watcher
func (c *PerformanceController) Watch() chan struct{} {
	performanceStop := make(chan struct{})
	err := c.Run(performanceStop)
	if err != nil {
		logrus.Fatal(err)
	}
	return performanceStop
}

func (c *PerformanceController) Run(stopCh chan struct{}) error {
	c.informerFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.podInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync")
	}
	return nil
}

func (c *PerformanceController) onPodAdd(obj interface{}) {
	pod := obj.(*api_v1.Pod)

	if c.podIsInConfig(pod) && pod.Status.Phase == api_v1.PodRunning {
		c.TestPod(pod)
	}
}

func (c *PerformanceController) onPodUpdate(old, new interface{}) {
	pod := new.(*api_v1.Pod)

	if c.podIsInConfig(pod) && pod.Status.Phase == api_v1.PodRunning {
		c.TestPod(pod)
	}
}

func (c *PerformanceController) TestPod(pod *api_v1.Pod) {
	for _, container := range pod.Spec.Containers {
		c.target.Test(c.httpClient, getName(pod, container))
	}
}

func getName(pod *api_v1.Pod, container api_v1.Container) string {
	name := fmt.Sprintf("%s.%s", pod.ObjectMeta.Labels["app"], container.Name)
	return name
}

func (c *PerformanceController) podIsInConfig(pod *api_v1.Pod) bool {
	// If it is in the apps array, return true
	for _, v := range c.config.Performance.Apps {
		if pod.ObjectMeta.Labels["app"] == v {
			return true
		}
	}
	return false
}
