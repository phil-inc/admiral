package logs

import (
	"testing"
	"time"

	"github.com/phil-inc/admiral/pkg/backend"
	"github.com/phil-inc/admiral/pkg/state"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var mocked_pod = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "hello-world",
		Namespace: "hello-world",
		Annotations: map[string]string{
			"admiral.io/ignore-containers": "world",
		},
	},
	Spec: v1.PodSpec{
		Containers: []v1.Container{
			{
				Name: "world",
			},
		},
	},
}

func Test_Handlers(t *testing.T) {
	st := state.New("hello-world")
	ignoreContainerAnnotation := ("admiral.io/ignore-containers")
	podFilterAnnotation := "test"
	rawLogChannel := make(chan backend.RawLog)

	log_watcher := New().State(st).PodFilterAnnotation(podFilterAnnotation).RawLogChannel(rawLogChannel).IgnoreContainerAnnotation(ignoreContainerAnnotation).Build()

	stateKey := "hello-world.hello-world.world"

	log_watcher.Add(mocked_pod)
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, "", st.Get(stateKey))

	mocked_pod.Status.Phase = v1.PodRunning
	log_watcher.Add(mocked_pod)
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, "", st.Get(stateKey))

	mocked_pod.ObjectMeta.Annotations["admiral.io/ignore-containers"] = "hello"
	mocked_pod.ObjectMeta.Annotations[podFilterAnnotation] = "world"
	log_watcher.Add(mocked_pod)
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, state.RUNNING, st.Get(stateKey))

	st.Delete(stateKey)
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, "", st.Get(stateKey))

	log_watcher.Update(mocked_pod, mocked_pod)
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, state.RUNNING, st.Get(stateKey))

	mocked_pod.Status.Phase = v1.PodFailed
	log_watcher.Update(mocked_pod, mocked_pod)
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, state.FINISHED, st.Get(stateKey))

	mocked_pod.ObjectMeta.Annotations["admiral.io/ignore-containers"] = "world"
	mocked_pod.Status.Phase = v1.PodSucceeded
	log_watcher.Update(mocked_pod, mocked_pod)
	time.Sleep(1 * time.Millisecond)
	log_watcher.Delete(mocked_pod)
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, state.FINISHED, st.Get(stateKey))

	mocked_pod.ObjectMeta.Annotations["admiral.io/ignore-containers"] = "hello"
	log_watcher.Delete(mocked_pod)
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, "", st.Get(stateKey))
}
