package realtime

import (
	"context"

	zerotrace "github.com/zeromicro/go-zero/core/trace"
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
