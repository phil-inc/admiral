package metrics_handlers

import "time"

type MetricBatch struct {
	Nodes map[string]NodeMetrics
	Pods  map[string]PodMetrics
}

type NodeMetrics struct {
	Name   string
	Cpu    Metric
	Memory Metric
}

type PodMetrics struct {
	Name           string
	Namespace      string
	Cpu            Metric
	Memory         Metric
	RXBytesTotal   Metric
	RXErrTotal     Metric
	RXPktDropTotal Metric
	RXPktTotal     Metric
	TXBytesTotal   Metric
	TXErrTotal     Metric
	TXPktDropTotal Metric
	TXPktTotal     Metric
	Containers     map[string]ContainerMetrics
}

type ContainerMetrics struct {
	Name      string
	Namespace string
	Cpu       Metric
	Memory    Metric
}

type Metric struct {
	Value     uint64
	Timestamp time.Time
}

type NetworkLabel struct {
	Container   string `json:"container"`
	ID          string `json:"id"`
	Image       string `json:"image"`
	Interface   string `json:"interface"`
	ContainerID string `json:"name"`
	Namespace   string `json:"namespace"`
	Pod         string `json:"pod"`
}
