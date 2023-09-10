package logs

import (
	"strings"

	"github.com/phil-inc/admiral/pkg/backend"
	"github.com/phil-inc/admiral/pkg/state"
	"github.com/phil-inc/admiral/pkg/stream/logstream"
	"github.com/phil-inc/admiral/pkg/utils"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

type logs struct {
	state                     *state.SharedMutable
	ignoreContainerAnnotation string
	podFilterAnnotation       string
	rawLogChannel             chan backend.RawLog
}

type builder struct {
	state                     *state.SharedMutable
	ignoreContainerAnnotation string
	podFilterAnnotation       string
	rawLogChannel             chan backend.RawLog
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

func (b *builder) PodFilterAnnotation(annotation string) *builder {
	b.podFilterAnnotation = annotation
	return b
}

func (b *builder) RawLogChannel(rawLogChannel chan backend.RawLog) *builder {
	b.rawLogChannel = rawLogChannel
	return b
}

func (b *builder) Build() *logs {
	return &logs{
		state:                     b.state,
		ignoreContainerAnnotation: b.ignoreContainerAnnotation,
		podFilterAnnotation:       b.podFilterAnnotation,
		rawLogChannel:             b.rawLogChannel,
	}
}

func (l *logs) Add(obj interface{}) {
	pod := obj.(*v1.Pod)

	if _, ok := pod.Annotations[l.podFilterAnnotation]; !ok {
		return
	}

	// check if the pod is running
	if pod.Status.Phase != v1.PodRunning {
		return
	}

	l.addContainersToState(pod)
}

func (l *logs) Update(old, new interface{}) {
	pod := new.(*v1.Pod)

	if _, ok := pod.Annotations[l.podFilterAnnotation]; !ok {
		return
	}

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

	if _, ok := pod.Annotations[l.podFilterAnnotation]; !ok {
		return
	}

	l.deleteContainersInState(pod)
}

func (l *logs) addContainersToState(pod *v1.Pod) {
	ignoreList := pod.Annotations[l.ignoreContainerAnnotation]

	for _, container := range pod.Spec.Containers {

		if ignoreContainer(ignoreList, container) {
			continue
		}

		name := utils.GenerateUniqueContainerName(pod, container)

		if v := l.state.Get(name); v != "" {
			continue
		}

		logrus.Println("Adding to state")
		logrus.Printf("\tPod: %s", pod.Name)
		logrus.Printf("\tContainer: %s", container.Name)

		l.state.Set(name, state.RUNNING)

		if l.state.GetKubeClient() != nil {
			metadata := make(map[string]string)
			if pod.Labels != nil {
				metadata = pod.Labels
			}
			metadata["pod"] = pod.Name
			metadata["namespace"] = pod.Namespace

			go logstream.New().State(l.state).Pod(pod).Container(container).Metadata(metadata).RawLogChannel(l.rawLogChannel).Build().Stream()
		}
	}
}

func (l *logs) finishContainersInState(pod *v1.Pod) {
	ignoreList := pod.Annotations[l.ignoreContainerAnnotation]

	for _, container := range pod.Spec.Containers {

		if ignoreContainer(ignoreList, container) {
			continue
		}

		name := utils.GenerateUniqueContainerName(pod, container)
		l.state.Delete(name)

		logrus.Println("Finishing state")
		logrus.Printf("\tPod: %s", pod.Name)
		logrus.Printf("\tContainer: %s", container.Name)
	}
}

func (l *logs) deleteContainersInState(pod *v1.Pod) {
	ignoreList := pod.Annotations[l.ignoreContainerAnnotation]

	for _, container := range pod.Spec.Containers {

		if ignoreContainer(ignoreList, container) {
			continue
		}

		name := utils.GenerateUniqueContainerName(pod, container)
		l.state.Delete(name)

		logrus.Println("Setting state deleted")
		logrus.Printf("\t\tPod: %s", pod.Name)
		logrus.Printf("\tContainer: %s", container.Name)
	}
}

func ignoreContainer(ignoreList string, container v1.Container) bool {
	if strings.Contains(ignoreList, container.Name) {
		return true
	}

	return false
}
