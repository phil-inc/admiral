package state

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

type SharedMutable struct {
	mutex sync.Mutex
	cluster string
	initTimestamp time.Time
	objects map[string]string
	objectChannel chan request
	deleteChannel chan string
	errChannel chan error
	kubeClient kubernetes.Interface
}

// New() instantiates a SharedMutable state and
// opens the goroutine listening for setObjects
// requests. Admiral will treat this like a singleton.
func New(cluster string) *SharedMutable {
	s := &SharedMutable{
		cluster: cluster,
		initTimestamp: time.Now(),
		objects: make(map[string]string),
		objectChannel: make(chan request),
		deleteChannel: make(chan string),
	}
	go s.run()
	return s
}

type request struct {
	Key string
	Value string
}

func (s *SharedMutable) run() {
	go s.setHandler()
	go s.deletionHandler()
}

func (s *SharedMutable) setHandler() {
	for request := range s.objectChannel {
		s.mutex.Lock()
		s.objects[request.Key] = request.Value
		s.mutex.Unlock()
	}
}

func (s *SharedMutable) deletionHandler() {
	for key := range s.deleteChannel {
		s.mutex.Lock()
		delete(s.objects, key)
		s.mutex.Unlock()
	}
}

func (s *SharedMutable) errorHandler() {
	for err := range s.errChannel {
		logrus.Errorln(err)
	}
}

//InitTimestamp returns the timestamp of when the
// SharedMutable state was created.
func (s *SharedMutable) InitTimestamp() time.Time {
	return s.initTimestamp
}

// Cluster returns the name of the cluster
func (s *SharedMutable) Cluster() string {
	return s.cluster
}

// Set takes a key/value and sends it to the objects
// channel where run() adds it to the state.
func (s *SharedMutable) Set(k string, v string) {
	r := request{
		Key: k,
		Value: v,
	}
	s.objectChannel <- r
}

// Get returns a value for the given key.
func (s *SharedMutable) Get(k string) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.objects[k]
}

// Delete sends a key to the deletion channel where
// run() removes it from the state.
func (s *SharedMutable) Delete(k string) {
	s.deleteChannel <- k
}

// SetKubeClient sets the Kubernetes client.
func (s *SharedMutable) SetKubeClient(client kubernetes.Interface) {
	s.kubeClient = client
}

// GetKubeClient retrieves the Kubernetes client.
func (s *SharedMutable) GetKubeClient() kubernetes.Interface {
	return s.kubeClient
}

// SetErrChannel sets a shared error channel.
func (s *SharedMutable) SetErrChannel(errChannel chan error) {
	s.errChannel = errChannel
}

// Error sends an error to the error channel.
func (s *SharedMutable) Error(e error) {
	s.errChannel <- e
}
