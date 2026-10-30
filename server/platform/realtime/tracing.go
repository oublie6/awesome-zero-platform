package realtime

import (
	"context"

	zerotrace "github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// withMessageTrace uses go-zero's selected tracer/provider, not a separate
// OTLP exporter. A connection may last hours, but each message has its own
// bounded span, propagated through business processing and SQL/Redis calls.
// Never record message payloads or bearer tokens as tracing attributes.
func withMessageTrace(ctx context.Context, dispatch func(context.Context)) {
	messageCtx, span := zerotrace.TracerFromContext(ctx).Start(ctx, "realtime.message")
	defer span.End()
	dispatch(messageCtx)
}

// Only bounded protocol error codes enter traces; handler errors can contain
// private business data and are deliberately not recorded here.
func sendMessageError(ctx context.Context, connection *client, requestID, code, message string) {
	oteltrace.SpanFromContext(ctx).SetStatus(codes.Error, code)
	connection.sendError(requestID, code, message)
}
