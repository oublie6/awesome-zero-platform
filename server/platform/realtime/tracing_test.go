package realtime

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
