package events

import (
	"testing"

	"github.com/phil-inc/admiral/pkg/state"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

var mocked_event *v1.Event = &v1.Event{
	Reason:  "hello-world",
	Message: "hello-world",
	InvolvedObject: v1.ObjectReference{
		Name:      "hello-world",
		Namespace: "hello-world",
	},
}

func Test_AddHandler(t *testing.T) {
	shared_state := state.New("cluster-world")
	msgCh := make(chan string)
	matchingFilter := []string{"hello-world"}
	failingFilter := []string{"goodnight"}
	mocked_event.ObjectMeta.CreationTimestamp.Time = shared_state.InitTimestamp().Add(10)

	event_watcher := New().State(shared_state).Channel(msgCh).Filter(matchingFilter).Build()

	formattedMsg := event_watcher.formatMessage(mocked_event)
	go func() {
		for msg := range msgCh {
			assert.Equal(t, formattedMsg, msg)
		}
	}()
	defer close(msgCh)

	event_watcher.Add(mocked_event)

	failing_watcher := New().State(shared_state).Channel(msgCh).Filter(failingFilter).Build()

	failing_watcher.Add(mocked_event)
}
