package metrics_handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/prometheus/prometheus/model/textparse"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Metricstream struct {
	Finished bool
	closed   chan struct{}
	pod      *api_v1.Pod
	handler  MetricsHandler
	batch    MetricBatch
}

// These are labels exported on the metrics from cadvisor
var (
	nodeCpuUsageTotal          = []byte("node_cpu_usage_seconds_total")
	nodeMemUsageTotal          = []byte("node_memory_working_set_bytes")
	podCpuUsageTotal           = []byte("pod_cpu_usage_seconds_total")
	podMemUsageTotal           = []byte("pod_memory_working_set_bytes")
	containerCpuUsageTotal     = []byte("container_cpu_usage_seconds_total")
	containerMemUsageTotal     = []byte("container_memory_working_set_bytes")
	containerNetRXBytesTotal   = []byte("container_network_receive_bytes_total")
	containerNetRXErrTotal     = []byte("container_network_receive_errors_total")
	containerNetRXPktDropTotal = []byte("container_network_receive_packets_dropped_total")
	containerNetRXPktTotal     = []byte("container_network_receive_packets_total")
	containerNetTXBytesTotal   = []byte("container_network_transmit_bytes_total")
	containerNetTXErrTotal     = []byte("container_network_transmit_errors_total")
	containerNetTXPktDropTotal = []byte("container_network_transmit_packets_dropped_total")
	containerNetTXPktTotal     = []byte("container_network_transmit_packets_total")

	containerName = []byte(`container="`)
	podName       = []byte(`pod="`)
	namespace     = []byte(`namespace="`)
	nInterface    = []byte(`interface="`)
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
		endpoints := []string{"resource", "cadvisor"}
		logrus.Println("Streaming metrics from", m.pod.Name)
		for !m.Finished {
			time.Sleep(1 * time.Second)
			for _, p := range endpoints {
				path := fmt.Sprintf("/api/v1/nodes/%s/proxy/metrics/%s", m.pod.Spec.NodeName, p)

				res := mc.RESTClient().Get().RequestURI(path).Do(context.Background())

				metrics, err := res.Raw()
				if err != nil {
					logrus.Errorf("Failed raw'ing the metrics: %s", err)
				}

				m.decodeMetrics(metrics)
			}
			ch <- m.batch
		}

	}()
}
func (m *Metricstream) Finish() {
	m.Finished = true
}

func (m *Metricstream) Delete() {
	m.Finish()
	logrus.Printf("Metricstream deleted: %s.%s", m.pod.Namespace, m.pod.Name)
	close(m.closed)
}

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

	var (
		err   error
		entry textparse.Entry
	)

	parser, err := textparse.New(b, "")
	if err != nil {
		logrus.Errorf("textparse error: %s", err)
	}

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
		// Network Metrics
		case seriesMatchesName(series, containerNetRXBytesTotal):
			_, po, _ := parseLabels(series[len(containerNetRXBytesTotal):])
			m.parseNetRXBytesTotal(*timestamp, value, po, series[len(containerNetTXPktTotal):])
		case seriesMatchesName(series, containerNetRXErrTotal):
			_, po, _ := parseLabels(series[len(containerNetRXErrTotal):])
			m.parseNetRXErrTotal(*timestamp, value, po, series[len(containerNetTXPktTotal):])
		case seriesMatchesName(series, containerNetRXPktDropTotal):
			_, po, _ := parseLabels(series[len(containerNetRXPktDropTotal):])
			m.parseNetRXPktDropTotal(*timestamp, value, po, series[len(containerNetTXPktTotal):])
		case seriesMatchesName(series, containerNetRXPktTotal):
			_, po, _ := parseLabels(series[len(containerNetRXPktTotal):])
			m.parseNetRXPktTotal(*timestamp, value, po, series[len(containerNetTXPktTotal):])
		case seriesMatchesName(series, containerNetTXBytesTotal):
			_, po, _ := parseLabels(series[len(containerNetTXBytesTotal):])
			m.parseNetTXBytesTotal(*timestamp, value, po, series[len(containerNetTXPktTotal):])
		case seriesMatchesName(series, containerNetTXErrTotal):
			_, po, _ := parseLabels(series[len(containerNetTXErrTotal):])
			m.parseNetTXErrTotal(*timestamp, value, po, series[len(containerNetTXPktTotal):])
		case seriesMatchesName(series, containerNetTXPktDropTotal):
			_, po, _ := parseLabels(series[len(containerNetTXPktDropTotal):])
			m.parseNetTXPktDropTotal(*timestamp, value, po, series[len(containerNetTXPktTotal):])
		case seriesMatchesName(series, containerNetTXPktTotal):
			_, po, _ := parseLabels(series[len(containerNetTXPktTotal):])
			m.parseNetTXPktTotal(*timestamp, value, po, series[len(containerNetTXPktTotal):])
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

func (m *Metricstream) parseNetRXBytesTotal(ts int64, value float64, name string, labels []byte) {
	networkInterfaceName := parseNetworkInterface(labels)

	if networkInterfaceName == "eth0" {
		p := m.batch.Pods[m.pod.Name]

		// convert millisecond to nanosecond
		p.RXBytesTotal.Timestamp = time.Unix(0, ts*1e6)
		// already nanoseconds
		p.RXBytesTotal.Value = uint64(value)
		m.batch.Pods[m.pod.Name] = p
	}
}

func (m *Metricstream) parseNetRXErrTotal(ts int64, value float64, name string, labels []byte) {
	networkInterfaceName := parseNetworkInterface(labels)

	if networkInterfaceName == "eth0" {
		p := m.batch.Pods[m.pod.Name]
		// convert millisecond to nanosecond
		p.RXErrTotal.Timestamp = time.Unix(0, ts*1e6)
		// already nanoseconds
		p.RXErrTotal.Value = uint64(value)
		m.batch.Pods[m.pod.Name] = p
	}
}

func (m *Metricstream) parseNetRXPktDropTotal(ts int64, value float64, name string, labels []byte) {
	networkInterfaceName := parseNetworkInterface(labels)

	if networkInterfaceName == "eth0" {
		p := m.batch.Pods[m.pod.Name]
		// convert millisecond to nanosecond
		p.RXPktDropTotal.Timestamp = time.Unix(0, ts*1e6)
		// already nanoseconds
		p.RXPktDropTotal.Value = uint64(value)
		m.batch.Pods[m.pod.Name] = p
	}
}

func (m *Metricstream) parseNetRXPktTotal(ts int64, value float64, name string, labels []byte) {
	networkInterfaceName := parseNetworkInterface(labels)

	if networkInterfaceName == "eth0" {
		p := m.batch.Pods[m.pod.Name]
		// convert millisecond to nanosecond
		p.RXPktTotal.Timestamp = time.Unix(0, ts*1e6)
		// already nanoseconds
		p.RXPktTotal.Value = uint64(value)
		m.batch.Pods[m.pod.Name] = p
	}
}

func (m *Metricstream) parseNetTXBytesTotal(ts int64, value float64, name string, labels []byte) {
	networkInterfaceName := parseNetworkInterface(labels)

	if networkInterfaceName == "eth0" {
		p := m.batch.Pods[m.pod.Name]
		// convert millisecond to nanosecond
		p.TXBytesTotal.Timestamp = time.Unix(0, ts*1e6)
		// already nanoseconds
		p.TXBytesTotal.Value = uint64(value)
		m.batch.Pods[m.pod.Name] = p
	}
}

func (m *Metricstream) parseNetTXErrTotal(ts int64, value float64, name string, labels []byte) {
	networkInterfaceName := parseNetworkInterface(labels)

	if networkInterfaceName == "eth0" {
		p := m.batch.Pods[m.pod.Name]
		// convert millisecond to nanosecond
		p.TXErrTotal.Timestamp = time.Unix(0, ts*1e6)
		// already nanoseconds
		p.TXErrTotal.Value = uint64(value)
		m.batch.Pods[m.pod.Name] = p
	}
}

func (m *Metricstream) parseNetTXPktDropTotal(ts int64, value float64, name string, labels []byte) {
	networkInterfaceName := parseNetworkInterface(labels)

	if networkInterfaceName == "eth0" {
		p := m.batch.Pods[m.pod.Name]
		// convert millisecond to nanosecond
		p.TXPktDropTotal.Timestamp = time.Unix(0, ts*1e6)
		// already nanoseconds
		p.TXPktDropTotal.Value = uint64(value)
		m.batch.Pods[m.pod.Name] = p
	}
}

func (m *Metricstream) parseNetTXPktTotal(ts int64, value float64, name string, labels []byte) {

	networkInterfaceName := parseNetworkInterface(labels)

	if networkInterfaceName == "eth0" {
		p := m.batch.Pods[m.pod.Name]
		// convert millisecond to nanosecond
		p.TXPktTotal.Timestamp = time.Unix(0, ts*1e6)
		// already nanoseconds
		p.TXPktTotal.Value = uint64(value)
		m.batch.Pods[m.pod.Name] = p
	}
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

func parseNetworkInterface(labels []byte) string {

	i := bytes.Index(labels, nInterface) + len(nInterface)
	j := bytes.IndexByte(labels[i:], '"')
	n := string(labels[i : i+j])

	return n
}
