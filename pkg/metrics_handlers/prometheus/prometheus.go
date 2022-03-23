package prometheus

import (
	"fmt"
	"net/http"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/metrics_handlers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Prometheus struct{}

// Init binds any settings from the config to the handler
func (p *Prometheus) Init(c *config.Config) error {
	return nil
}

// Handle transforms & exposes the metrics for Prometheus scraping
func (p *Prometheus) Handle(metrics <-chan metrics_handlers.MetricBatch) {
	g := make(map[string]prometheus.Gauge)
	r := prometheus.NewRegistry()
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
				r.MustRegister(g[cpuGuage])

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
				r.MustRegister(g[memGuage])
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
				r.MustRegister(g[cpuGuage])

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
				r.MustRegister(g[memGuage])

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
					r.MustRegister(g[cpuGuage])

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
					r.MustRegister(g[memGuage])
				}
			}
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
	http.ListenAndServe(":2112", nil)
}
