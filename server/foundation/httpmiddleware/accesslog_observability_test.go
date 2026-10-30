package httpmiddleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/handler"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestAccessLogEmitsSafeStructuredFields(t *testing.T) {
	var lines bytes.Buffer
	previous := logx.Reset()
	logx.SetWriter(logx.NewWriter(&lines))
	defer func() {
		logx.Reset()
		if previous != nil {
			logx.SetWriter(previous)
		}
	}()

	wrapped := RequestID(RequestIDConfig{HeaderName: "X-Request-Id", MaxLength: 64})(
		AccessLog()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})),
	)
	req := httptest.NewRequest(http.MethodPost, "/doudizhu/rooms?access_token=must-not-log", nil)
	req.Header.Set("X-Request-Id", "log-observability-test")
	req.Header.Set("Authorization", "Bearer secret-must-not-log")
	wrapped.ServeHTTP(httptest.NewRecorder(), req)

	if bytes.Contains(lines.Bytes(), []byte("must-not-log")) {
		t.Fatalf("sensitive query/header included in log: %s", lines.String())
	}
	var got map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(lines.Bytes()), &got); err != nil {
		t.Fatalf("log json: %v; data=%s", err, lines.String())
	}
	for name, want := range map[string]any{
		"content":     "http access",
		"eventAction": "http.access",
		"module":      "doudizhu",
		"requestId":   "log-observability-test",
		"statusCode":  float64(201),
	} {
		if got[name] != want {
			t.Fatalf("%s = %#v, want %#v", name, got[name], want)
		}
	}
	if _, ok := got["durationMs"]; !ok {
		t.Fatalf("durationMs missing: %s", lines.String())
	}
}

func TestOuterAccessLogCorrelatesGoZeroRouteTrace(t *testing.T) {
	var lines bytes.Buffer
	previous := logx.Reset()
	logx.SetWriter(logx.NewWriter(&lines))
	defer func() {
		logx.Reset()
		if previous != nil {
			logx.SetWriter(previous)
		}
	}()
	provider := sdktrace.NewTracerProvider()
	defer provider.Shutdown(context.Background())
	previousProvider, previousPropagator := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer func() { otel.SetTracerProvider(previousProvider); otel.SetTextMapPropagator(previousPropagator) }()
	wrapped := AccessLog()(handler.TraceHandler("test", "/health/live")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	response := httptest.NewRecorder()
	wrapped.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	ctx := (propagation.TraceContext{}).Extract(context.Background(), propagation.HeaderCarrier(response.Header()))
	traceID := oteltrace.SpanContextFromContext(ctx).TraceID()
	if !traceID.IsValid() {
		t.Fatal("go-zero did not produce response trace context")
	}
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(lines.Bytes()), &record); err != nil {
		t.Fatal(err)
	}
	if record["trace"] != traceID.String() {
		t.Fatalf("access log lost route trace: %v", record["trace"])
	}
}
