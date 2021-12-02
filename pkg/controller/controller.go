package controller

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/event"
	"github.com/phil-inc/admiral/pkg/handlers"
	"github.com/phil-inc/admiral/pkg/utils"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const maxRetries = 5

var serverStartTime time.Time

// An event from the API server
type Event struct {
	key          string
	eventType    string
	namespace    string
	resourceType string
}

// Our controller
type Controller struct {
	logger       *logrus.Entry
	clientset    kubernetes.Interface
	queue        workqueue.RateLimitingInterface
	informer     cache.SharedIndexInformer
	eventHandler handlers.Handler
}

// Start creates watchers and runs their controllers
func Start(conf *config.Config, eventHandler handlers.Handler) {
	var kubeClient kubernetes.Interface

	if _, err := rest.InClusterConfig(); err != nil {
		kubeClient = utils.GetClientOutOfCluster()
	} else {
		kubeClient = utils.GetClient()
	}

	// Unhealthy pod informer
	unhealthyPodInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = "involvedObject.kind=Pod,type=Warning,reason=Failed"
				return kubeClient.CoreV1().Events(conf.Namespace).List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = "involvedObject.kind=Pod,type=Warning,reason=Failed"
				return kubeClient.CoreV1().Events(conf.Namespace).Watch(context.Background(), options)
			},
		},
		&api_v1.Event{},
		0,
		cache.Indexers{},
	)

	unhealthyPodController := newResourceController(kubeClient, eventHandler, unhealthyPodInformer, "Unhealthy")
	stopUnhealthyPodCh := make(chan struct{})
	defer close(stopUnhealthyPodCh)

	go unhealthyPodController.Run(stopUnhealthyPodCh)

	// NodeNotReady informer
	nodeNotReadyInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = "involvedObject.kind=Node,type=Normal,reason=NodeNotReady"
				return kubeClient.CoreV1().Events(conf.Namespace).List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = "involvedObject.kind=Node,type=Normal,reason=NodeNotReady"
				return kubeClient.CoreV1().Events(conf.Namespace).Watch(context.Background(), options)
			},
		},
		&api_v1.Event{},
		0,
		cache.Indexers{},
	)

	nodeNotReadyController := newResourceController(kubeClient, eventHandler, nodeNotReadyInformer, "NodeNotReady")
	stopNodeNotReadyCh := make(chan struct{})
	defer close(stopNodeNotReadyCh)

	go nodeNotReadyController.Run(stopNodeNotReadyCh)

	// BackOff informer
	backoffInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = "involvedObject.kind=Pod,type=Warning,reason=BackOff"
				return kubeClient.CoreV1().Events(conf.Namespace).List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = "involvedObject.kind=Pod,type=Warning,reason=BackOff"
				return kubeClient.CoreV1().Events(conf.Namespace).Watch(context.Background(), options)
			},
		},
		&api_v1.Event{},
		0,
		cache.Indexers{},
	)

	backoffController := newResourceController(kubeClient, eventHandler, backoffInformer, "Backoff")
	stopBackoffCh := make(chan struct{})
	defer close(stopBackoffCh)

	go backoffController.Run(stopBackoffCh)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}

func newResourceController(client kubernetes.Interface, eventHandler handlers.Handler, informer cache.SharedIndexInformer, resourceType string) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	var newEvent Event
	var err error
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newEvent.key, err = cache.MetaNamespaceKeyFunc(obj)
			newEvent.eventType = "create"
			newEvent.resourceType = resourceType
			logrus.WithField("pkg", "admiral-"+resourceType).Infof("Processing add to %v: %s", resourceType, newEvent.key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			newEvent.key, err = cache.MetaNamespaceKeyFunc(old)
			newEvent.eventType = "update"
			newEvent.resourceType = resourceType
			logrus.WithField("pkg", "admiral-"+resourceType).Infof("Processing update to %v: %s", resourceType, newEvent.key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
		DeleteFunc: func(obj interface{}) {
			newEvent.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			newEvent.eventType = "delete"
			newEvent.resourceType = resourceType
			newEvent.namespace = utils.GetObjectMetaData(obj).Namespace
			logrus.WithField("pkg", "admiral-"+resourceType).Infof("Processing delete to %v: %s", resourceType, newEvent.key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
	})

	return &Controller{
		logger:       logrus.WithField("pkg", "admiral-"+resourceType),
		clientset:    client,
		informer:     informer,
		queue:        queue,
		eventHandler: eventHandler,
	}
}

// Run starts a given controller.
// stopCh is a channel sending the interrupt signal to stop the controller.
func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.logger.Info("Starting Admiral controller")
	serverStartTime = time.Now().Local()

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Cache sync timeout"))
		return
	}

	c.logger.Info("Admiral synced and ready")
	// Loop indefinitely
	// .Until restarts the worker after one second
	wait.Until(c.runWorker, time.Second, stopCh)
}

// HasSynced is required by cache.Controller interface
func (c *Controller) HasSynced() bool {
	return c.informer.HasSynced()
}

// runWorker will loop on processNextItem
func (c *Controller) runWorker() {
	for c.processNextItem() {

	}
}

// processNextItem acts on one key from queue
func (c *Controller) processNextItem() bool {
	// Grab the next key in queue
	newEvent, quit := c.queue.Get()
	if quit {
		return false
	}

	// Always tell the queue the key is being worked upon
	defer c.queue.Done(newEvent)

	// Work on the key
	err := c.processItem(newEvent.(Event))

	if err == nil {
		// Remove the key from the queue
		c.queue.Forget(newEvent)
	} else if c.queue.NumRequeues(newEvent) < maxRetries {
		// Error within the retry limit
		c.logger.Errorf("Error processing %s (retry): %v", newEvent.(Event).key, err)
		// Requeue the item
		c.queue.AddRateLimited(newEvent.(Event).key)
	} else {
		// Error w/ too many retries
		c.logger.Errorf("Error processing %s (too many retries): %v", newEvent.(Event).key, err)
		c.queue.Forget(newEvent)
		utilruntime.HandleError(err)
	}
	return true
}

func (c *Controller) processItem(newEvent Event) error {
	c.logger.Infof("Processing: %s", newEvent.key)

	obj, _, err := c.informer.GetIndexer().GetByKey(newEvent.key)
	if err != nil {
		return fmt.Errorf("Error fetching object with key %s from store: %v", newEvent.key, err)
	}

	objectMetadata := utils.GetObjectMetaData(obj)

	// If the namespace is empty
	if newEvent.namespace == "" && strings.Contains(newEvent.key, "/") {
		substring := strings.Split(newEvent.key, "/")
		newEvent.namespace = substring[0]
		newEvent.key = substring[1]
	}

	// Do nothing if the event happened before the server started
	if serverStartTime.After(objectMetadata.CreationTimestamp.Time) {
		return nil
	}

	// Do something based on the event type
	kbEvent := event.Event{
		Namespace: newEvent.namespace,
		Kind:      newEvent.resourceType,
	}

	switch newEvent.eventType {
	case "create": //update, delete
		kbEvent.Name = objectMetadata.Name
	default:
		return nil
	}
	c.eventHandler.Handle(kbEvent)
	return nil
}
