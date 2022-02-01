package logs

import (
	"testing"
	"time"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/logstores"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

func mockLogController() *LogController {
	mockClient := &fake.Clientset{}
	mockConfig := &config.Config{}
	mockLogstore := &logstores.Default{}
	mockInformerFactory := informers.NewSharedInformerFactory(mockClient, 30*time.Second)

	return NewLogController(mockInformerFactory, mockClient, mockConfig, mockLogstore)
}

func Test_NewLogController(t *testing.T) {
	c := mockLogController()
	stopWatch := c.Watch()
	defer close(stopWatch)
}

// func Test_onPodEvents(t *testing.T) {
// 	c := mockLogController()

// 	var tcs = []struct {
// 		name string
// 		pod  *api_v1.Pod
// 	}{
// 		{"Pod in running phase", &api_v1.Pod{Status: api_v1.PodStatus{Phase: api_v1.PodRunning}}},
// 		{"Pod in not running phase", &api_v1.Pod{Status: api_v1.PodStatus{Phase: api_v1.PodFailed}}},
// 	}

// 	for _, tt := range tcs {
// 		t.Run(tt.name, func(*testing.T) {
// 			c.onPodAdd(tt.pod)
// 			c.onPodUpdate(&api_v1.Pod{Status: api_v1.PodStatus{Phase: api_v1.PodPending}}, tt.pod)
// 			c.onPodDelete(tt.pod)
// 		})
// 	}
// }
