package logs

import (
	"fmt"
	"strings"

	"github.com/phil-inc/admiral/pkg/backend"
	"github.com/phil-inc/admiral/pkg/state"
	"github.com/phil-inc/admiral/pkg/stream/logstream"
	v1 "k8s.io/api/core/v1"
)

type logs struct {
	state *state.SharedMutable
	ignoreContainerAnnotation string
	rawLogChannel chan backend.RawLog	
}

type builder struct {
	state *state.SharedMutable
	ignoreContainerAnnotation string
	rawLogChannel chan backend.RawLog	
}

func New() *builder {
	return &builder{}
}

func (b *builder) State(state *state.SharedMutable) *builder {
	b.state = state	
	return b
}

func (b *builder) IgnoreContainerAnnotation(annotation string) *builder {
	b.ignoreContainerAnnotation = annotation 
	return b
}

func (b *builder) RawLogChannel(rawLogChannel chan backend.RawLog) *builder {
	b.rawLogChannel = rawLogChannel
	return b
}

func (b *builder) Build() *logs {
	return &logs{
		state: b.state,
		ignoreContainerAnnotation: b.ignoreContainerAnnotation,
		rawLogChannel: b.rawLogChannel,
	}
}

func (l *logs) Add(obj interface{}) {
	pod := obj.(*v1.Pod)

	// check if the pod is running
	if pod.Status.Phase != v1.PodRunning {
		return
	}

	l.addContainersToState(pod)
}

func (l *logs) Update(old, new interface{}) {
	pod := new.(*v1.Pod)

	// check if the pod is running
	if pod.Status.Phase == v1.PodRunning {
		l.addContainersToState(pod)
	}

	// check if the pod is finishing
	if pod.Status.Phase == v1.PodSucceeded ||
	   pod.Status.Phase == v1.PodFailed {
		l.finishContainersInState(pod)
	}
}

func (l *logs) Delete(obj interface{}) {
	pod := obj.(*v1.Pod)

	l.deleteContainersInState(pod)
}

func (l *logs) addContainersToState(pod *v1.Pod) {
	ignoreList := pod.Annotations[l.ignoreContainerAnnotation]

	for _, container := range pod.Spec.Containers {

		if ignoreContainer(ignoreList, container) {
			continue
		}

		name := generateUniqueContainerName(pod, container)	
		l.state.Set(name, state.RUNNING)
		go logstream.New().State(l.state).RawLogChannel(l.rawLogChannel).Build().Stream()
	}
}

func (l *logs) finishContainersInState(pod *v1.Pod) {
	ignoreList := pod.Annotations[l.ignoreContainerAnnotation]

	for _, container := range pod.Spec.Containers {

		if ignoreContainer(ignoreList, container) {
			continue
		}

		name := generateUniqueContainerName(pod, container)
		l.state.Set(name, state.FINISHED)
	}
}

func (l *logs) deleteContainersInState(pod *v1.Pod) {
	ignoreList := pod.Annotations[l.ignoreContainerAnnotation]

	for _, container := range pod.Spec.Containers {

		if ignoreContainer(ignoreList, container) {
			continue
		}

		name := generateUniqueContainerName(pod, container)
		l.state.Delete(name)	
	}
}

func ignoreContainer(ignoreList string, container v1.Container) bool {
	if strings.Contains(ignoreList, container.Name) {
		return true
	}

	return false
}

func generateUniqueContainerName(pod *v1.Pod, container v1.Container) string {
	return fmt.Sprintf("%s.%s.%s", pod.ObjectMeta.Namespace, pod.Name, container.Name)
}