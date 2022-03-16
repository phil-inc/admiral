package metrics

import (
	"context"
	"fmt"

	"github.com/phil-inc/admiral/pkg/metrics_handlers"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

type metricstream struct {
	Finished bool
	closed   chan struct{}
	pod      string
	handler  metrics_handlers.MetricsHandler
}

func NewMetricStream(pod string, handler metrics_handlers.MetricsHandler) *metricstream {
	return &metricstream{
		Finished: false,
		closed:   make(chan struct{}),
		pod:      pod,
		handler:  handler,
	}
}

func (m *metricstream) Start(clientset *rest.Config) {
	mc, err := metrics.NewForConfig(clientset)
	if err != nil {
		logrus.Errorf("Failed converting to a metrics client: %s", err)
	}

	go func() {
		metricsWatcher, err := mc.MetricsV1beta1().PodMetricses(metav1.NamespaceAll).Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name=%s", m.pod)})
		if err != nil {
			logrus.Errorf("Failed getting metrics for %s: %s", m.pod, err)
		}

		go func() {
			<-m.closed
			metricsWatcher.Stop()
		}()

		for metrics := range metricsWatcher.ResultChan() {
			logrus.Print(metrics)
		}

		m.Finish()
	}()
}
func (m *metricstream) Finish() {
	m.Finished = true
}
func (m *metricstream) Delete() {
	close(m.closed)
}
