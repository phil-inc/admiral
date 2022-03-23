package metrics_handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Metricstream struct {
	Finished bool
	pod      *api_v1.Pod
	handler  MetricsHandler
	batch    MetricBatch
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

func NewMetricStream(pod *api_v1.Pod, handler MetricsHandler) *Metricstream {
	return &Metricstream{
		Finished: false,
		pod:      pod,
	}
}

func (m *Metricstream) Start(r *rest.Config, ch chan<- MetricBatch) {
	go func() {
		mc, err := metrics_client.NewForConfig(r)
		if err != nil {
			logrus.Errorf("Failed creating metrics client: %s", err)
		}

		path := fmt.Sprintf("/api/v1/nodes/%s/proxy/metrics/resource", m.pod.Spec.NodeName)

		res := mc.RESTClient().Get().RequestURI(path).Do(context.Background())

		metrics, err := res.Raw()
		if err != nil {
			logrus.Errorf("Failed raw'ing the metrics: %s", err)
		}

		m.decodeMetrics(metrics)

		ch <- m.batch

		m.Finish()

	}()
}
func (m *Metricstream) Finish() {
	m.Finished = true
}

func (m *Metricstream) Delete() {}

func (m *Metricstream) decodeMetrics(b []byte) {
	// Label the node & pod metrics with names and namespaces
	m.batch = MetricBatch{
		Nodes: map[string]NodeMetrics{
			m.pod.Spec.NodeName: {
				Name: m.pod.Spec.NodeName,
			},
		},
		Pods: map[string]PodMetrics{
			m.pod.Name: {
				Name:       m.pod.Name,
				Namespace:  m.pod.Namespace,
				Containers: make(map[string]ContainerMetrics),
			},
		},
	}

	// Label the container metrics with names and namespaces
	for _, container := range m.pod.Spec.Containers {
		m.batch.Pods[m.pod.Name].Containers[container.Name] = ContainerMetrics{
			Name:      container.Name,
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
			_, _, c := parseLabels(series[len(containerCpuUsageTotal):])
			m.parseContainerCpuUsage(*timestamp, value, c)

		case seriesMatchesName(series, containerMemUsageTotal):
			// select the container matching the labels
			_, _, c := parseLabels(series[len(containerMemUsageTotal):])
			m.parseContainerMemUsage(*timestamp, value, c)

		default:
			continue
		}
	}
}

func (m *Metricstream) parseNodeCpuUsage(ts int64, value float64) {
	n := m.batch.Nodes[m.pod.Spec.NodeName]
	// convert second to nanosecond
	n.Cpu.Value = uint64(value * 1e9)
	// convert millisecond to nanosecond
	n.Cpu.Timestamp = time.Unix(0, ts*1e6)
	m.batch.Nodes[m.pod.Spec.NodeName] = n
}

func (m *Metricstream) parsePodCpuUsage(ts int64, value float64) {
	p := m.batch.Pods[m.pod.Name]
	// convert second to nanosecond
	p.Cpu.Value = uint64(value * 1e9)
	// convert millisecond to nanosecond
	p.Cpu.Timestamp = time.Unix(0, ts*1e6)
	m.batch.Pods[m.pod.Name] = p
}

func (m *Metricstream) parseContainerCpuUsage(ts int64, value float64, name string) {
	c := m.batch.Pods[m.pod.Name].Containers[name]
	// convert second to nanosecond
	c.Cpu.Value = uint64(value * 1e9)
	// convert millisecond to nanosecond
	c.Cpu.Timestamp = time.Unix(0, ts*1e6)
	m.batch.Pods[m.pod.Name].Containers[name] = c
}

func (m *Metricstream) parseNodeMemUsage(ts int64, value float64) {
	n := m.batch.Nodes[m.pod.Spec.NodeName]
	// convert millisecond to nanosecond
	n.Memory.Timestamp = time.Unix(0, ts*1e6)
	// already nanoseconds
	n.Memory.Value = uint64(value)
	m.batch.Nodes[m.pod.Spec.NodeName] = n
}

func (m *Metricstream) parsePodMemUsage(ts int64, value float64) {
	p := m.batch.Pods[m.pod.Name]
	// convert millisecond to nanosecond
	p.Memory.Timestamp = time.Unix(0, ts*1e6)
	// already nanoseconds
	p.Memory.Value = uint64(value)
	m.batch.Pods[m.pod.Name] = p
}

func (m *Metricstream) parseContainerMemUsage(ts int64, value float64, name string) {
	c := m.batch.Pods[m.pod.Name].Containers[name]
	// convert millisecond to nanosecond
	c.Memory.Timestamp = time.Unix(0, ts*1e6)
	// already nanoseconds
	c.Memory.Value = uint64(value)
	m.batch.Pods[m.pod.Name].Containers[name] = c
}

func seriesMatchesName(s []byte, n []byte) bool {
	return bytes.HasPrefix(s, n) && (len(s) == len(n) || s[len(n)] == '{')
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
