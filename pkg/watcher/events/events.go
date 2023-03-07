package events

import (
	"fmt"

	"github.com/phil-inc/admiral/pkg/state"
	v1 "k8s.io/api/core/v1"
)

type events struct {
	state *state.SharedMutable
	channel chan string
	filter []string
}

type builder struct {
	state *state.SharedMutable
	channel chan string 
	filter []string
}

func New() *builder {
	return &builder{}
}

// State sets the SharedMutable state.
func (b *builder) State(state *state.SharedMutable) *builder {
	b.state = state
	return b
}

// Channel sets a string channel that should forward to the backend.
func (b *builder) Channel(channel chan string) *builder {
	b.channel = channel
	return b
}

// Filter sets a slice of strings where each element
// is a type of event that we will forward to the backend.
func (b *builder) Filter(filter []string) *builder {
	b.filter = filter
	return b
}

func (b *builder) Build() *events {
	return &events{
		state: b.state,
		channel: b.channel,
		filter: b.filter,
	}
}

// Add should be bound to a SharedInformer's EventListener.
// It will handle new cluster events as they are created and
// if they pass the filter, pass them to the backend channel.
func (e *events) Add(obj interface{}) {
	event := obj.(*v1.Event)

	// check if the event was created
	// before admiral started.
	if e.state.InitTimestamp().Before(event.ObjectMeta.CreationTimestamp.Time) {
		if e.inFilter(event.Message) {
			e.channel <- e.formatMessage(event)
		}	
	}
}

func (e *events) inFilter(s string) bool {
	for _, v := range e.filter {
		if s == v {
			return true
		}
	}
	return false
}

func (e *events) formatMessage(event *v1.Event) string {
	return fmt.Sprintf(`cluster: %s \n
						namespace: %s \n
						object: %s \n
						reason: %s \n
						message: %s \n
						timestamp: %s \n`,
	e.state.Cluster(), event.Namespace, event.InvolvedObject.Name, event.Reason, event.Message, event.CreationTimestamp.Time)
}

func Update(obj interface{}) {}

func Delete(obj interface{}) {}
