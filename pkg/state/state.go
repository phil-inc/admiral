package state

import "time"

type SharedMutable struct {
	initTimestamp time.Time
	objects map[string]string
	objectChannel chan Request
}

// New() instantiates a SharedMutable state and
// opens the goroutine listening for setObjects
// requests. Admiral will treat this like a singleton.
func New() *SharedMutable {
	s := &SharedMutable{
		initTimestamp: time.Now(),
		objects: make(map[string]string),
		objectChannel: make(chan Request),
	}
	go s.run()
	return s
}

type Request struct {
	Key string
	Value string
}

func (s *SharedMutable) run() {
	for request := range s.objectChannel {
		s.objects[request.Key] = request.Value
	}
}

//InitTimestamp returns the timestamp of when the
// SharedMutable state was created.
func (s *SharedMutable) InitTimestamp() time.Time {
	return s.initTimestamp
}
