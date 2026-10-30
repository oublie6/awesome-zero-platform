package mysqlstore

import (
	"context"
	"errors"
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type commandCleanupRecorder struct {
	*tracetest.SpanRecorder
	script *script
	t      *testing.T
}

func (r *commandCleanupRecorder) OnEnd(span sdktrace.ReadOnlySpan) {
	if span.Name() == "sql.transaction" {
		r.script.mu.Lock()
		remaining := len(r.script.steps)
		r.script.mu.Unlock()
		if remaining != 0 {
			r.t.Errorf("command transaction span ended with %d SQL/cleanup operations pending", remaining)
		}
	}
	r.SpanRecorder.OnEnd(span)
}

func TestCommandTracePreservesReadCommittedAndOutcome(t *testing.T) {
	for _, outcome := range []string{"success", "begin", "callback", "commit", "panic", "retryable"} {
		t.Run(outcome, func(t *testing.T) {
			failure := errors.New("command transaction failure")
			steps := []scriptStep{beginStep()}
			switch outcome {
			case "begin":
				steps[0].err = failure
			case "success":
				steps = append(steps, commitStep())
			case "commit":
				step := commitStep()
				step.err = failure
				steps = append(steps, step)
			default:
				steps = append(steps, rollbackStep())
			}
			// The existing scripted driver rejects any isolation other than READ COMMITTED.
			db, script := newScriptDB(t, steps...)
			store, err := New(scriptDatabase{db})
			if err != nil {
				t.Fatal(err)
			}
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(&commandCleanupRecorder{SpanRecorder: recorder, script: script, t: t}))
			defer provider.Shutdown(context.Background())
			ctx, parent := provider.Tracer("test").Start(context.Background(), "message")
			defer parent.End()
			var got error
			var recovered any
			var observed oteltrace.SpanContext
			func() {
				defer func() { recovered = recover() }()
				got = store.WithinCommand(ctx, "actor", "command", func(ctx context.Context, _ application.Transaction) error {
					observed = oteltrace.SpanContextFromContext(ctx)
					switch outcome {
					case "panic":
						panic(failure)
					case "callback":
						return failure
					case "retryable":
						return errors.New("Error 1213: Deadlock found when trying to get lock")
					}
					return nil
				})
			}()
			switch outcome {
			case "success":
				if got != nil || recovered != nil {
					t.Fatalf("success changed: %v %v", got, recovered)
				}
			case "panic":
				if recovered != failure {
					t.Fatalf("panic identity changed: %v", recovered)
				}
			case "retryable":
				if !errors.Is(got, application.ErrRetryableTransaction) {
					t.Fatalf("retryable classification changed: %v", got)
				}
			default:
				if !errors.Is(got, failure) || recovered != nil {
					t.Fatalf("error identity changed: %v %v", got, recovered)
				}
			}
			if outcome != "begin" && (!observed.IsValid() || observed.TraceID() != parent.SpanContext().TraceID() || observed.SpanID() == parent.SpanContext().SpanID()) {
				t.Fatal("command callback lost child context")
			}
			spans := recorder.Ended()
			if len(spans) != 1 || spans[0].Name() != "sql.transaction" || spans[0].Parent().SpanID() != parent.SpanContext().SpanID() || (spans[0].Status().Code == codes.Error) != (outcome != "success") {
				t.Fatal("command transaction span parent/outcome incorrect")
			}
			script.assertDone(t)
		})
	}
}
