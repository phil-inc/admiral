package controller

import (
	"fmt"
	"context"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/event"
	"github.com/phil-inc/admiral/pkg/handlers"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"github.com/sirupsen/logrus"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/kubernetes"
)

type EventController struct {
	informerFactory informers.SharedInformerFactory
	eventInformer   coreinformers.EventInformer
	handler         handlers.Handler
	config          *config.Config
	clientset       kubernetes.Interface
}

// Instantiates a controller for watching and handling events
func NewEventController(informerFactory informers.SharedInformerFactory, handler handlers.Handler, config *config.Config, clientset kubernetes.Interface) *EventController {
	eventInformer := informerFactory.Core().V1().Events()

	c := &EventController{
		informerFactory: informerFactory,
		eventInformer:   eventInformer,
		handler: handler,
		config: config,
		clientset: clientset,
	}

	eventInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onEventAdd,
			UpdateFunc: c.onEventUpdate,
			DeleteFunc: c.onEventDelete,
		},
	)

	return c
}

func (c *EventController) Run(stopCh chan struct{}) error {
	c.informerFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.eventInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync")
	}
	return nil
}

// when an event object is created
func (c *EventController) onEventAdd(obj interface{}) {
	e := obj.(*api_v1.Event)

	if serverStartTime.After(e.ObjectMeta.CreationTimestamp.Time) {
		return
	}

	switch e.Reason {
	case "NodeNotReady", "Unhealthy":
		c.handler.Handle(c.newSendableEvent(e))
	}
}

// when an event object is updated
func (c *EventController) onEventUpdate(old, new interface{}) {}

// when an event object is deleted
func (c *EventController) onEventDelete(obj interface{}) {}

func (c *EventController) getLabelFromNode(key string, node string) string {
	s, err := c.clientset.CoreV1().Nodes().Get(context.Background(), node, meta_v1.GetOptions{})
	if err != nil {
		logrus.Errorf("failed getting node: %s", err)
	}
	return s.ObjectMeta.Labels[key]
}

func (c *EventController) newSendableEvent(e *api_v1.Event) (n event.Event) {
	n.Namespace = e.ObjectMeta.Namespace
	n.Reason    = e.Reason
	n.Cluster   = c.config.Cluster
	n.Name      = e.ObjectMeta.Name
	n.Extra     = fmt.Sprintf("%s - %s", e.Message, e.ObjectMeta.CreationTimestamp.Time)

	if c.config.Fargate && e.InvolvedObject.Kind == "Node" {
		p := trimNodeName(e.ObjectMeta.Name)
		n.Extra = fmt.Sprintf("%s - %s", n.Extra, c.getLabelFromNode("topology.kubernetes.io/zone", p))
	}
	return
}