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
	Name       string
	Namespace  string
	Cpu        Metric
	Memory     Metric
	Containers map[string]ContainerMetrics
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
