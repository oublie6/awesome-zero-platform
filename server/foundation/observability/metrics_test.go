package observability

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
