package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestMessageTraceIsShortLivedChildOfContext(t *testing.T) {
	provider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	defer func() { _ = provider.Shutdown(context.Background()) }()
	ctx, parent := provider.Tracer("test").Start(context.Background(), "parent")
	defer parent.End()
	var observed oteltrace.SpanContext
	withMessageTrace(ctx, func(ctx context.Context) {
		observed = oteltrace.SpanFromContext(ctx).SpanContext()
	})
	if !observed.IsValid() || observed.TraceID() != parent.SpanContext().TraceID() ||
		observed.SpanID() == parent.SpanContext().SpanID() {
		t.Fatalf("message span must be a distinct child of existing context: %v", observed)
	}
}

func TestMessageFailureMarksSpanWithoutLeakingHandlerError(t *testing.T) {
	for _, kind := range []string{"invalid", "unknown", "handler", "success"} {
		t.Run(kind, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			defer provider.Shutdown(context.Background())
			ctx, parent := provider.Tracer("test").Start(context.Background(), "request")
			defer parent.End()
			h, err := New(Config{Enabled: true}, testAuthenticator{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			const secret = "private handler payload"
			if err := h.RegisterHandler("test.failure", func(context.Context, ConnectionContext, Envelope) (*Envelope, error) { return nil, errors.New(secret) }); err != nil {
				t.Fatal(err)
			}
			connection := newClient(h, nil, "test-connection", Identity{})
			defer connection.cancel()
			envelope := Envelope{ID: "request", Type: TypeSystemPing}
			switch kind {
			case "invalid":
				envelope.Type = ""
			case "unknown":
				envelope.Type = "test.unknown"
			case "handler":
				envelope.Type = "test.failure"
			}
			withMessageTrace(ctx, func(ctx context.Context) { h.handle(ctx, connection, envelope) })
			spans := recorder.Ended()
			if len(spans) != 1 {
				t.Fatalf("ended spans=%d", len(spans))
			}
			span := spans[0]
			if kind == "success" {
				if span.Status().Code == codes.Error {
					t.Fatal("successful message reported as error")
				}
			} else if span.Status().Code != codes.Error {
				t.Fatalf("message failure not reported: %v", span.Status())
			}
			if strings.Contains(span.Status().Description, secret) || len(span.Events()) != 0 {
				t.Fatal("private handler error entered trace")
			}
			var payload []byte
			select {
			case payload = <-connection.send:
			default:
				t.Fatal("message response was not queued")
			}
			var response Envelope
			if err := json.Unmarshal(payload, &response); err != nil {
				t.Fatal(err)
			}
			if response.ID != envelope.ID {
				t.Fatal("response request ID changed")
			}
			if kind == "success" {
				if response.Type != TypeSystemPong {
					t.Fatal("ping behavior changed")
				}
			} else if response.Type != TypeSystemError {
				t.Fatal("error response behavior changed")
			}
		})
	}
}
