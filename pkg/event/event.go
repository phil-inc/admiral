package event

import "fmt"

type Event struct {
	Namespace string
	Kind      string
	Component string
	Host      string
	Reason    string
	Status    string
	Name      string
	Cluster   string
}

// Message returns event message
// These correlate to the informers defined in controller.go
func (e *Event) Message() string {
	msg := fmt.Sprintf("%s %s/%s: `%s`",
		e.Cluster,
		e.Namespace,
		e.Name,
		e.Kind,
	)
	return msg
}
