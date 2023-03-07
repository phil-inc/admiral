package events

import (
	"testing"

	"github.com/phil-inc/admiral/pkg/state"
	v1 "k8s.io/api/core/v1"
)

var mocked_event *v1.Event = &v1.Event{
	Reason: "hello-world",
	Message: "hello-world",
	InvolvedObject: v1.ObjectReference{
		Name: "hello-world",
		Namespace: "hello-world",
	},
}

func Test_AddHandler(t *testing.T) {
	shared_state := state.New("cluster-world")
	msgCh := make(chan string)
	matchingFilter := []string{"hello-world"}
	failingFilter := []string{"goodnight"}

	event_watcher := New().State(shared_state).Channel(msgCh).Filter(matchingFilter).Build()

	event_watcher.Add(mocked_event)
	
	failing_watcher := New().State(shared_state).Channel(msgCh).Filter(failingFilter).Build()

	failing_watcher.Add(mocked_event)
}
