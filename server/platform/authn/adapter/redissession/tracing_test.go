package redissession

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRotationFailureSpanPreservesParentAndError(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer provider.Shutdown(context.Background())
	ctx, parent := provider.Tracer("test").Start(context.Background(), "refresh")
	defer parent.End()
	failure := errors.New("dial failed")
	client := redis.NewClient(&redis.Options{
		Addr: "unused:6379", MaxRetries: -1,
		Dialer: func(context.Context, string, string) (net.Conn, error) { return nil, failure },
	})
	defer client.Close()
	store, err := New(client, "test:")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Rotate(ctx, "session", "current", "next", time.Now().Add(time.Minute)); !errors.Is(err, failure) {
		t.Fatalf("error identity changed: %v", err)
	}
	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans=%d", len(spans))
	}
	span := spans[0]
	if span.Name() != "redis.watch.rotate" || span.Status().Code != codes.Error || len(span.Events()) == 0 {
		t.Fatalf("missing rotation failure: %v", span.Status())
	}
	if span.Parent().SpanID() != parent.SpanContext().SpanID() || span.SpanContext().TraceID() != parent.SpanContext().TraceID() {
		t.Fatal("rotation lost parent")
	}
}
