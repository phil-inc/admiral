package main

import (
	"github.com/phil-inc/admiral/pkg/watcher"
	"k8s.io/client-go/tools/cache"
)

func InitWatcher(w watcher.Watcher, i cache.SharedIndexInformer) error {
	i.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    w.Add,
			UpdateFunc: w.Update,
			DeleteFunc: w.Delete,
		},
	)
	return nil
}
