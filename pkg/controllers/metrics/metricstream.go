package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/phil-inc/admiral/pkg/metrics_handlers"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

type metricstream struct {
	Finished bool
	closed   chan struct{}
	pod      *api_v1.Pod
	handler  metrics_handlers.MetricsHandler
	batch    *MetricBatch
}

// These are labels exported on the metrics from cadvisor
var (
	nodeCpuUsageTotal      = []byte("node_cpu_usage_seconds_total")
	nodeMemUsageTotal      = []byte("node_memory_working_set_bytes")
	podCpuUsageTotal       = []byte("pod_cpu_usage_seconds_total")
	podMemUsageTotal       = []byte("pod_memory_working_set_bytes")
	containerCpuUsageTotal = []byte("container_cpu_usage_seconds_total")
	containerMemUsageTotal = []byte("container_memory_working_set_bytes")
	containerName          = []byte(`container="`)
	podName                = []byte(`pod="`)
	namespace              = []byte(`namespace="`)
)

func NewMetricStream(pod *api_v1.Pod, handler metrics_handlers.MetricsHandler) *metricstream {
	return &metricstream{
		Finished: false,
		closed:   make(chan struct{}),
		pod:      pod,
		handler:  handler,
	}
}

func (m *metricstream) Start(r *rest.Config) {
	go func() {
		mc, err := metrics_client.NewForConfig(r)
		if err != nil {
			logrus.Errorf("Failed creating metrics client: %s", err)
		}

		path := fmt.Sprintf("/api/v1/nodes/%s/proxy/metrics/resource", m.pod.Spec.NodeName)

		res := mc.RESTClient().Get().RequestURI(path).Do(context.Background())

		go func() {
			<-m.closed
			// stream.Close()
		}()

		metties, err := res.Raw()
		if err != nil {
			logrus.Errorf("Failed raw'ing the metrics: %s", err)
		}

		logrus.Printf(string(metties))
		// metrics := bufio.NewScanner(stream)

		// for metrics.Scan() {
		// 	logrus.Print(metrics.Text())
		// }

		m.Finish()
	}()
}
func (m *metricstream) Finish() {
	m.Finished = true
}
func (m *metricstream) Delete() {
	close(m.closed)
}

func (m *metricstream) decodeMetrics(b []byte) {
	// Label the node & pod metrics with names and namespaces
	// This really assumes there's only 1 pod on the node (fargate use-case)
	m.batch = &MetricBatch{
		nodes: map[string]NodeMetrics{
			m.pod.Spec.NodeName: NodeMetrics{},
		},
		pods: map[string]PodMetrics{
			m.pod.Name: PodMetrics{
				Namespace: m.pod.Namespace,
			},
		},
	}

	// Label the container metrics with names and namespaces
	for _, container := range m.pod.Spec.Containers {
		m.batch.pods[m.pod.Name].Containers[container.Name] = ContainerMetrics{
			Namespace: m.pod.Namespace,
		}
	}

	parser := textparse.New(b, "")

	var (
		err   error
		entry textparse.Entry
	)

	for {
		if entry, err = parser.Next(); err != nil {
			if err == io.EOF {
				break
			}
		}

		if entry != textparse.EntrySeries {
			continue
		}

		series, timestamp, value := parser.Series()

		// match a timeseries to one of the exported labels
		// if it's a match, parse its value
		switch {
		case seriesMatchesName(series, nodeCpuUsageTotal):
			m.parseNodeCpuUsage(*timestamp, value)

		case seriesMatchesName(series, nodeMemUsageTotal):
			m.parseNodeMemUsage(*timestamp, value)

		case seriesMatchesName(series, podCpuUsageTotal):
			m.parsePodCpuUsage(*timestamp, value)

		case seriesMatchesName(series, podMemUsageTotal):
			m.parsePodMemUsage(*timestamp, value)

		case seriesMatchesName(series, containerCpuUsageTotal):
			// select the container matching the labels
			// ns, p, c := parseLabels(series[len(containerCpuUsageTotal):])

		case seriesMatchesName(series, containerMemUsageTotal):
			// select the container matching the labels
			// ns, p, c := parseLabels(series[len(containerMemUsageTotal):])

		default:
			continue
		}
	}
}

func (m *metricstream) parseNodeCpuUsage(ts int64, value float64) {
	n := m.batch.nodes[m.pod.Spec.NodeName]

	// convert second to nanosecond
	n.Cpu.Value = uint64(value * 1e9)

	// convert millisecond to nanosecond
	n.Cpu.Timestamp = time.Unix(0, ts*1e6)

	m.batch.nodes[m.pod.Spec.NodeName] = n
}

func (m *metricstream) parsePodCpuUsage(ts int64, value float64) {
	p := m.batch.pods[m.pod.Name]

	// convert second to nanosecond
	p.Cpu.Value = uint64(value * 1e9)

	// convert millisecond to nanosecond
	p.Cpu.Timestamp = time.Unix(0, ts*1e6)

	m.batch.pods[m.pod.Name] = p
}

// func (m *metricstream) parseContainerCpuUsage(ts int64, value float64) {
// 	// convert second to nanosecond
// 	met.Value = uint64(value * 1e9)

// 	// convert millisecond to nanosecond
// 	met.Timestamp = time.Unix(0, ts*1e6)
// }

func (m *metricstream) parseNodeMemUsage(ts int64, value float64) {
	n := m.batch.nodes[m.pod.Spec.NodeName]
	// convert millisecond to nanosecond
	n.Memory.Timestamp = time.Unix(0, ts*1e6)

	// already nanoseconds
	n.Memory.Value = uint64(value)

	m.batch.nodes[m.pod.Spec.NodeName] = n
}

func (m *metricstream) parsePodMemUsage(ts int64, value float64) {
	p := m.batch.pods[m.pod.Name]
	// convert millisecond to nanosecond
	p.Memory.Timestamp = time.Unix(0, ts*1e6)

	// already nanoseconds
	p.Memory.Value = uint64(value)

	m.batch.pods[m.pod.Name] = p
}

// func (m *metricstream) parseContainerMemUsage(ts int64, value float64, met *Metric) {
// 	// convert millisecond to nanosecond
// 	met.Timestamp = time.Unix(0, ts*1e6)

// 	// already nanoseconds
// 	met.Value = uint64(value)
// }

func seriesMatchesName(s, n []byte) bool {
	return bytes.HasPrefix(s, n) && (s[len(n)] == '{' || len(s) == len(n))
}

func parseLabels(labels []byte) (ns string, pod string, container string) {
	i := bytes.Index(labels, containerName) + len(containerName)
	j := bytes.IndexByte(labels[i:], '"')
	container = string(labels[i : i+j])

	i = bytes.Index(labels, podName) + len(podName)
	j = bytes.IndexByte(labels[i:], '"')
	pod = string(labels[i : i+j])

	i = bytes.Index(labels, namespace) + len(namespace)
	j = bytes.IndexByte(labels[i:], '"')
	ns = string(labels[i : i+j])

	return
}
