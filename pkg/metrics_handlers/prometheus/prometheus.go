package prometheus

import (
	"fmt"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/metrics_handlers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/sirupsen/logrus"
)

type Prometheus struct{}

var conf *config.Config

// Init binds any settings from the config to the handler
func (p *Prometheus) Init(c *config.Config) error {
	conf = c
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
				cpuGauge := fmt.Sprintf("%s_cpu", n.Name)
				if _, exists := g[cpuGauge]; !exists {
					g[cpuGauge] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "node_cpu",
						ConstLabels: prometheus.Labels{
							"name": n.Name,
						},
					})
				}
				g[cpuGauge].Set(float64(n.Cpu.Value))
				pushToGateway(n.Name, "node_cpu", g[cpuGauge])

				memGauge := fmt.Sprintf("%s_mem", n.Name)
				if _, exists := g[memGauge]; !exists {
					g[memGauge] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "node_memory",
						ConstLabels: prometheus.Labels{
							"name": n.Name,
						},
					})
				}
				g[memGauge].Set(float64(n.Memory.Value))
				pushToGateway(n.Name, "node_memory", g[memGauge])
			}

			for _, po := range m.Pods {
				cpuGauge := fmt.Sprintf("%s_cpu", po.Name)
				if _, exists := g[cpuGauge]; !exists {
					g[cpuGauge] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "pod_cpu",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[cpuGauge].Set(float64(po.Cpu.Value))
				pushToGateway(po.Name, "pod_cpu", g[cpuGauge])

				memGauge := fmt.Sprintf("%s_mem", po.Name)
				if _, exists := g[memGauge]; !exists {
					g[memGauge] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "pod_memory",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[memGauge].Set(float64(po.Memory.Value))
				pushToGateway(po.Name, "pod_memory", g[memGauge])

				// Network counters - not available on containers themselves via
				// Using Gauge instead of Counter to Set() the value since Counter type doesn't have ability to set an arbitrary value
				netRXBytesCounter := fmt.Sprintf("%s_container_network_receive_bytes_total", po.Name)
				if _, exists := g[netRXBytesCounter]; !exists {
					g[netRXBytesCounter] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "container_network_receive_bytes_total",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"pod":       po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[netRXBytesCounter].Set(float64(po.RXBytesTotal.Value))
				pushToGateway(po.Name, "container_network_receive_bytes_total", g[netRXBytesCounter])

				containerNetRXErrCounter := fmt.Sprintf("%s_container_network_receive_errors_total", po.Name)
				if _, exists := g[containerNetRXErrCounter]; !exists {
					g[containerNetRXErrCounter] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "container_network_receive_errors_total",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"pod":       po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[containerNetRXErrCounter].Set(float64(po.RXErrTotal.Value))
				pushToGateway(po.Name, "container_network_receive_errors_total", g[containerNetRXErrCounter])

				containerNetRXPktDropCounter := fmt.Sprintf("%s_container_network_receive_packets_dropped_total", po.Name)
				if _, exists := g[containerNetRXPktDropCounter]; !exists {
					g[containerNetRXPktDropCounter] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "container_network_receive_packets_dropped_total",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"pod":       po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[containerNetRXPktDropCounter].Set(float64(po.RXPktDropTotal.Value))
				pushToGateway(po.Name, "container_network_receive_packets_dropped_total", g[containerNetRXPktDropCounter])

				containerNetRXPktCounter := fmt.Sprintf("%s_container_network_receive_packets_total", po.Name, po.Name)
				if _, exists := g[containerNetRXPktCounter]; !exists {
					g[containerNetRXPktCounter] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "container_network_receive_packets_total",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"pod":       po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[containerNetRXPktCounter].Set(float64(po.RXPktTotal.Value))
				pushToGateway(po.Name, "container_network_receive_packets_total", g[containerNetRXPktCounter])

				netTXBytesCounter := fmt.Sprintf("%s_container_network_transmit_bytes_total", po.Name)
				if _, exists := g[netTXBytesCounter]; !exists {
					g[netTXBytesCounter] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "container_network_transmit_bytes_total",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"pod":       po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[netTXBytesCounter].Set(float64(po.TXBytesTotal.Value))
				pushToGateway(po.Name, "container_network_transmit_bytes_total", g[netTXBytesCounter])

				containerNetTXErrCounter := fmt.Sprintf("%s_container_network_transmit_errors_total", po.Name)
				if _, exists := g[containerNetTXErrCounter]; !exists {
					g[containerNetTXErrCounter] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "container_network_transmit_errors_total",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"pod":       po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[containerNetTXErrCounter].Set(float64(po.TXErrTotal.Value))
				pushToGateway(po.Name, "container_network_transmit_errors_total", g[containerNetTXErrCounter])

				containerNetTXPktDropCounter := fmt.Sprintf("%s_container_network_transmit_packets_dropped_total", po.Name)
				if _, exists := g[containerNetTXPktDropCounter]; !exists {
					g[containerNetTXPktDropCounter] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "container_network_transmit_packets_dropped_total",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"pod":       po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[containerNetTXPktDropCounter].Set(float64(po.TXPktDropTotal.Value))
				pushToGateway(po.Name, "container_network_transmit_packets_dropped_total", g[containerNetTXPktDropCounter])

				containerNetTXPktCounter := fmt.Sprintf("%s_container_network_transmit_packets_total", po.Name)
				if _, exists := g[containerNetTXPktCounter]; !exists {
					g[containerNetTXPktCounter] = promauto.NewGauge(prometheus.GaugeOpts{
						Name: "container_network_transmit_packets_total",
						ConstLabels: prometheus.Labels{
							"name":      po.Name,
							"pod":       po.Name,
							"namespace": po.Namespace,
						},
					})
				}
				g[containerNetTXPktCounter].Set(float64(po.TXPktTotal.Value))
				pushToGateway(po.Name, "container_network_transmit_packets_total", g[containerNetTXPktCounter])

				for _, c := range po.Containers {
					cpuGauge := fmt.Sprintf("%s_%s_cpu", c.Name, po.Name)
					if _, exists := g[cpuGauge]; !exists {
						g[cpuGauge] = promauto.NewGauge(prometheus.GaugeOpts{
							Name: "container_cpu",
							ConstLabels: prometheus.Labels{
								"name":      c.Name,
								"pod":       po.Name,
								"namespace": c.Namespace,
							},
						})
					}
					g[cpuGauge].Set(float64(c.Cpu.Value))
					pushToGateway(c.Name, "container_cpu", g[cpuGauge])

					memGauge := fmt.Sprintf("%s_%s_mem", c.Name, po.Name)
					if _, exists := g[memGauge]; !exists {
						g[memGauge] = promauto.NewGauge(prometheus.GaugeOpts{
							Name: "container_memory",
							ConstLabels: prometheus.Labels{
								"name":      c.Name,
								"pod":       po.Name,
								"namespace": c.Namespace,
							},
						})
					}
					g[memGauge].Set(float64(c.Memory.Value))
					pushToGateway(c.Name, "container_memory", g[memGauge])
				}
			}
		}
	}()
}

func pushToGateway(name, metrictype string, g prometheus.Gauge) error { //g prometheus.Gauge
	pg := conf.Metrics.Handler.PushGateway
	if len(name) > 0 {
		if err := push.New(pg, name).
			Collector(g).
			Grouping("metric", metrictype).
			Push(); err != nil {
			logrus.Errorf("Error pushing metrics | METRIC, HOST, NAME: %s %s %s", metrictype, pg, name)
			return err
		}
	}

	return nil
}
