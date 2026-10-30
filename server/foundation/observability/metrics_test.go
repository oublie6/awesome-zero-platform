package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedisNativePoolMetricsPreservePrometheusHandler(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	m := NewMetrics("example")
	m.requests.WithLabelValues("GET", "/health/live", "200").Inc()
	if err := m.RegisterRedisPool(client); err != nil {
		t.Fatalf("register Redis pool: %v", err)
	}
	record := httptest.NewRecorder()
	m.Handler().ServeHTTP(record, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if record.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", record.Code)
	}
	body := record.Body.String()
	for _, name := range []string{
		"example_redis_native_pool_total_connections",
		"example_redis_native_pool_idle_connections",
		"example_redis_native_pool_stale_connections",
		"example_http_requests_total",
	} {
		if !strings.Contains(body, name) {
			t.Errorf("metrics scrape lacks %s", name)
		}
	}
}
