package observability

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/redis/go-redis/v9"
	zeroprom "github.com/zeromicro/go-zero/core/prometheus"
)

type Metrics struct {
	namespace string
	registry  *prometheus.Registry
	requests  *prometheus.CounterVec
	duration  *prometheus.HistogramVec
}

func NewMetrics(namespace string) *Metrics {
	// Enable go-zero's existing operation instruments without starting its
	// separate HTTP agent. The application serves the selected families below.
	zeroprom.Enable()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_requests_total",
		Help:      "Total HTTP requests completed by method, path, and status.",
	}, []string{"method", "path", "status"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration in seconds by method and path.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		requests,
		duration,
	)
	return &Metrics{namespace: namespace, registry: registry, requests: requests, duration: duration}
}

// RegisterRedisPool exports native-client pool health without sensitive Redis
// command arguments or keys. The managed go-zero client has its own internal
// pool and this metric MUST NOT be interpreted as its total connection count.
func (m *Metrics) RegisterRedisPool(client *redis.Client) error {
	if m == nil || client == nil {
		return nil
	}
	for _, gauge := range []struct {
		name  string
		help  string
		value func() float64
	}{
		{name: "redis_native_pool_total_connections", help: "Connections in the native Redis pool (excludes go-zero managed Redis).", value: func() float64 { return float64(client.PoolStats().TotalConns) }},
		{name: "redis_native_pool_idle_connections", help: "Idle native Redis pool connections.", value: func() float64 { return float64(client.PoolStats().IdleConns) }},
		{name: "redis_native_pool_stale_connections", help: "Stale connections removed from the native Redis pool.", value: func() float64 { return float64(client.PoolStats().StaleConns) }},
	} {
		if err := m.registry.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: m.namespace, Name: gauge.name, Help: gauge.help,
		}, gauge.value)); err != nil {
			return err
		}
	}
	return nil
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(prometheus.Gatherers{
		m.registry,
		redisMetricsGatherer{source: prometheus.DefaultGatherer},
	}, promhttp.HandlerOpts{})
}

// go-zero v1.10.2 registers its managed Redis operation/pool metrics globally.
// Select only those verified families: merging the entire global registry
// would duplicate Go/process collectors and expose unrelated instruments.
type redisMetricsGatherer struct{ source prometheus.Gatherer }

func (g redisMetricsGatherer) Gather() ([]*dto.MetricFamily, error) {
	families, err := g.source.Gather()
	if err != nil {
		return nil, err
	}
	selected := make([]*dto.MetricFamily, 0, 10)
	for _, family := range families {
		switch family.GetName() {
		case "redis_client_requests_duration_ms", "redis_client_requests_error_total", "redis_client_requests_slow_total",
			"redis_client_pool_hit_total", "redis_client_pool_miss_total", "redis_client_pool_timeout_total",
			"redis_client_pool_conn_total_current", "redis_client_pool_conn_idle_current",
			"redis_client_pool_conn_stale_total", "redis_client_pool_conn_max":
			selected = append(selected, family)
		}
	}
	return selected, nil
}

func (m *Metrics) Registerer() prometheus.Registerer {
	if m == nil {
		return nil
	}
	return m.registry
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	if m == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		path := boundedMetricPath(r.URL.Path)
		method := boundedMetricMethod(r.Method)
		m.requests.WithLabelValues(method, path, strconv.Itoa(recorder.status)).Inc()
		m.duration.WithLabelValues(method, path).Observe(time.Since(started).Seconds())
	})
}

// HTTP permits extension methods supplied by clients. Keep those from creating
// unlimited time series even for requests rejected by the router.
func boundedMetricMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
		return method
	default:
		return "OTHER"
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	connection, buffered, err := hijacker.Hijack()
	if err == nil {
		w.status = http.StatusSwitchingProtocols
	}
	return connection, buffered, err
}

func (w *statusRecorder) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// boundedMetricPath limits Prometheus label cardinality. Arbitrary user-provided
// URLs must never produce unbounded time-series labels.
func boundedMetricPath(path string) string {
	switch path {
	case "/auth/login", "/auth/logout", "/auth/refresh", "/auth/session",
		"/health/live", "/health/ready", "/metrics", "/ws":
		return path
	}
	switch {
	case strings.HasPrefix(path, "/admin/"):
		return "/admin/:resource"
	case strings.HasPrefix(path, "/doudizhu/") || strings.HasPrefix(path, "/v1/doudizhu/"):
		return "/doudizhu/:resource"
	default:
		return "/other"
	}
}
