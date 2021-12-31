package controller

import (
	"fmt"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/event"
	"github.com/phil-inc/admiral/pkg/handlers"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type EventController struct {
	informerFactory informers.SharedInformerFactory
	eventInformer   coreinformers.EventInformer
	handler         handlers.Handler
	config          *config.Config
}

// Instantiates a controller for watching and handling events
func NewEventController(informerFactory informers.SharedInformerFactory) *EventController {
	eventInformer := informerFactory.Core().V1().Events()

	c := &EventController{
		informerFactory: informerFactory,
		eventInformer:   eventInformer,
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
		c.handler.Handle(event.Event{
			Namespace: e.ObjectMeta.Namespace,
			Kind:      e.Reason,
			Cluster:   c.config.Cluster,
			Name:      e.ObjectMeta.Name,
			Extra:     e.Message,
		})
	}
}

// when an event object is updated
func (c *EventController) onEventUpdate(old, new interface{}) {}

// when an event object is deleted
func (c *EventController) onEventDelete(obj interface{}) {}
