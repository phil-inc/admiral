package metrics

import "time"

type MetricBatch struct {
	nodes map[string]NodeMetrics
	pods  map[string]PodMetrics
}

type NodeMetrics struct {
	Cpu    Metric
	Memory Metric
}

type PodMetrics struct {
	Namespace  string
	Cpu        Metric
	Memory     Metric
	Containers map[string]ContainerMetrics
}

type ContainerMetrics struct {
	Namespace string
	Cpu       Metric
	Memory    Metric
}

type Metric struct {
	Value     uint64
	Timestamp time.Time
}
