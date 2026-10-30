package observability

import (
	"io"
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

func TestMetricsRecordsRequestsAndExposesPrometheusFormat(t *testing.T) {
	metrics := NewMetrics("test_platform")
	handler := metrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	request := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	metricsResponse := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(metricsResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body, err := io.ReadAll(metricsResponse.Result().Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	text := string(body)
	for _, expected := range []string{
		`test_platform_http_requests_total{method="POST",path="/auth/login",status="201"} 1`,
		`test_platform_http_request_duration_seconds_count{method="POST",path="/auth/login"} 1`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("metrics output does not contain %q:\n%s", expected, text)
		}
	}
}

func TestBoundedMetricPath(t *testing.T) {
	for _, item := range []struct{ path, want string }{
		{"/auth/login", "/auth/login"},
		{"/admin/accounts/one-user", "/admin/:resource"},
		{"/admin/accounts/another-user", "/admin/:resource"},
		{"/doudizhu/hands/hand-123", "/doudizhu/:resource"},
		{"/v1/doudizhu/hands/id-123", "/doudizhu/:resource"},
		{"/random/sensitive-identifier", "/other"},
	} {
		if got := boundedMetricPath(item.path); got != item.want {
			t.Errorf("boundedMetricPath(%q)=%q, want %q", item.path, got, item.want)
		}
	}
}

func TestClientExtensionMethodsHaveBoundedMetricCardinality(t *testing.T) {
	metrics := NewMetrics("bounded")
	handler := metrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	for _, method := range []string{"CUSTOM_A", "CUSTOM_B", "CUSTOM_C", http.MethodGet} {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, "/arbitrary", nil))
	}
	response := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := response.Body.String()
	for _, want := range []string{
		`bounded_http_requests_total{method="OTHER",path="/other",status="405"} 3`,
		`bounded_http_request_duration_seconds_count{method="OTHER",path="/other"} 3`,
		`bounded_http_requests_total{method="GET",path="/other",status="405"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing bounded metric %s", want)
		}
	}
	if strings.Contains(body, "CUSTOM_") {
		t.Fatal("client-controlled method entered metric labels")
	}
}
