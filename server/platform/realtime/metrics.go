package realtime

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

type metricsRecorder interface {
	connectionAccepted()
	connectionRejected(string)
	connectionClosed()
	messageReceived()
	messageSent()
	slowConsumer()
}

type noopMetrics struct{}

func (noopMetrics) connectionAccepted()        {}
func (noopMetrics) connectionRejected(string) {}
func (noopMetrics) connectionClosed()          {}
func (noopMetrics) messageReceived()           {}
func (noopMetrics) messageSent()               {}
func (noopMetrics) slowConsumer()              {}

type PrometheusMetrics struct {
	active           prometheus.Gauge
	accepted         prometheus.Counter
	rejected         *prometheus.CounterVec
	messagesReceived prometheus.Counter
	messagesSent     prometheus.Counter
	slowConsumers    prometheus.Counter
}

func NewPrometheusMetrics(registerer prometheus.Registerer, namespace string) (*PrometheusMetrics, error) {
	if registerer == nil {
		return nil, fmt.Errorf("prometheus registerer is required")
	}
	metrics := &PrometheusMetrics{
		active: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "realtime",
			Name:      "active_connections",
			Help:      "Current authenticated WebSocket connections.",
		}),
		accepted: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "realtime",
			Name:      "connections_accepted_total",
			Help:      "Authenticated WebSocket connections accepted.",
		}),
		rejected: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "realtime",
			Name:      "connections_rejected_total",
			Help:      "WebSocket handshakes or registrations rejected by bounded reason.",
		}, []string{"reason"}),
		messagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "realtime",
			Name:      "messages_received_total",
			Help:      "Valid JSON WebSocket messages received.",
		}),
		messagesSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "realtime",
			Name:      "messages_sent_total",
			Help:      "JSON WebSocket messages queued for delivery.",
		}),
		slowConsumers: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "realtime",
			Name:      "slow_consumer_disconnects_total",
			Help:      "Connections disconnected because their bounded send queue filled.",
		}),
	}
	if err := registerer.Register(metrics); err != nil {
		return nil, fmt.Errorf("register realtime metrics: %w", err)
	}
	return metrics, nil
}

func (m *PrometheusMetrics) Describe(channel chan<- *prometheus.Desc) {
	m.active.Describe(channel)
	m.accepted.Describe(channel)
	m.rejected.Describe(channel)
	m.messagesReceived.Describe(channel)
	m.messagesSent.Describe(channel)
	m.slowConsumers.Describe(channel)
}

func (m *PrometheusMetrics) Collect(channel chan<- prometheus.Metric) {
	m.active.Collect(channel)
	m.accepted.Collect(channel)
	m.rejected.Collect(channel)
	m.messagesReceived.Collect(channel)
	m.messagesSent.Collect(channel)
	m.slowConsumers.Collect(channel)
}

func (m *PrometheusMetrics) connectionAccepted() {
	m.active.Inc()
	m.accepted.Inc()
}

func (m *PrometheusMetrics) connectionRejected(reason string) {
	m.rejected.WithLabelValues(reason).Inc()
}

func (m *PrometheusMetrics) connectionClosed() {
	m.active.Dec()
}

func (m *PrometheusMetrics) messageReceived() {
	m.messagesReceived.Inc()
}

func (m *PrometheusMetrics) messageSent() {
	m.messagesSent.Inc()
}

func (m *PrometheusMetrics) slowConsumer() {
	m.slowConsumers.Inc()
}
