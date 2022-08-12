package prometheus

import (
	"fmt"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/metrics_handlers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/push"
)

type Prometheus struct{}

// Init binds any settings from the config to the handler
func (p *Prometheus) Init(c *config.Config) error {
	return nil
}

// Handle transforms & exposes the metrics for Prometheus scraping
func (p *Prometheus) Handle(metrics <-chan metrics_handlers.MetricBatch) {
	g := make(map[string]prometheus.Gauge)
	go func() {
		// This infinite loop breaks when the channel for passing in metrics closes
		// The receiver <-metrics also blocks until it is passed metrics
		for {
			m, open := <-metrics

			if !open {
				break
			}

			for _, n := range m.Nodes {
				cpuGuage := fmt.Sprintf("%s_cpu", n.Name)
				if _, exists := g[cpuGuage]; !exists {
					g[cpuGuage] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "node_cpu",
						ConstLabels: prometheus.Labels{
							"name": n.Name,
						},
					})
				}
				g[cpuGuage].Set(float64(n.Cpu.Value))
				pushToGateway(n.Name, "node_cpu", g[cpuGauge])

				memGuage := fmt.Sprintf("%s_mem", n.Name)
				if _, exists := g[memGuage]; !exists {
					g[memGuage] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "node_memory",
						ConstLabels: prometheus.Labels{
							"name": n.Name,
						},
					})
				}
				g[memGuage].Set(float64(n.Memory.Value))
				pushToGateway(n.Name, "node_memory", g[memGauge])
			}

			for _, po := range m.Pods {
				cpuGuage := fmt.Sprintf("%s_cpu", po.Name)
				if _, exists := g[cpuGuage]; !exists {
					g[cpuGuage] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "pod_cpu",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[cpuGuage].Set(float64(po.Cpu.Value))
				pushToGateway(po.Name, "pod_cpu", g[cpuGauge])

				memGuage := fmt.Sprintf("%s_mem", po.Name)
				if _, exists := g[memGuage]; !exists {
					g[memGuage] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "pod_memory",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[memGuage].Set(float64(po.Memory.Value))
				pushToGateway(po.Name, "pod_memory", g[memGauge])

				for _, c := range po.Containers {
					cpuGuage := fmt.Sprintf("%s_%s_cpu", c.Name, po.Name)
					if _, exists := g[cpuGuage]; !exists {
						g[cpuGuage] = promauto.NewGauge(prometheus.GaugeOpts{
							Name: "container_cpu",
							ConstLabels: prometheus.Labels{
								"name":      c.Name,
								"pod":       po.Name,
								"namespace": c.Namespace,
							},
						})
					}
					g[cpuGuage].Set(float64(c.Cpu.Value))
					pushToGateway(c.Name, "container_cpu", g[cpuGauge])

					memGuage := fmt.Sprintf("%s_%s_mem", c.Name, po.Name)
					if _, exists := g[memGuage]; !exists {
						g[memGuage] = promauto.NewGauge(prometheus.GaugeOpts{
							Name: "container_memory",
							ConstLabels: prometheus.Labels{
								"name":      c.Name,
								"pod":       po.Name,
								"namespace": c.Namespace,
							},
						})
					}
					g[memGuage].Set(float64(c.Memory.Value))
					pushToGateway(c.Name, "container_memory", g[memGauge])
				}
			}
		}
	}()
}

func pushToGateway(name, metrictype string, g prometheus.Gauge) error {
	if err := push.New("http://localhost:9091/", name).
		Collector(g).
		Grouping("metric", metrictype).
		Push(); err != nil {
		logrus.Errorf("%s %s", "Error pushing", metrictype, "metrics to pushGateway for", name)
		return err
	}
	return nil
}
