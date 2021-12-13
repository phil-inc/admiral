package event

import "fmt"

type Event struct {
	Namespace       string
	Kind            string
	Component       string
	Host            string
	Reason          string
	Status          string
	Name            string
	NodeDescription string
}

// Message returns event message
// These correlate to the informers defined in controller.go
func (e *Event) Message() string {
	msg := fmt.Sprintf("%s/%s: `%s`",
		e.Namespace,
		e.Name,
		e.Kind,
	)

	if e.NodeDescription != "" {
		msg = fmt.Sprintf("%s \n %s", msg, e.NodeDescription)
	}
	return msg
}
