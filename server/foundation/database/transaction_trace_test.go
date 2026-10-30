package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTransactionSpanReportsOutcomeAfterCleanup(t *testing.T) {
	for _, outcome := range []string{"success", "begin", "callback", "commit", "rollback", "panic"} {
		t.Run(outcome, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			defer provider.Shutdown(context.Background())
			ctx, parent := provider.Tracer("test").Start(context.Background(), "request")
			defer parent.End()
			const privateFailure = "private-transaction-callback-payload"
			failure := errors.New(privateFailure)
			db, mock := newMockDB(t)
			begin := mock.ExpectBegin()
			switch outcome {
			case "begin":
				begin.WillReturnError(failure)
			case "success":
				mock.ExpectCommit()
			case "commit":
				mock.ExpectCommit().WillReturnError(failure)
			case "rollback":
				mock.ExpectRollback().WillReturnError(failure)
			default:
				mock.ExpectRollback()
			}
			resource := &Resource{beginTxFn: db.BeginTx}
			var got error
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				got = resource.WithinTransaction(ctx, func(context.Context, *sql.Tx) error {
					if outcome == "panic" {
						panic(failure)
					}
					if outcome == "callback" || outcome == "rollback" {
						return failure
					}
					return nil
				})
			}()
			if outcome == "panic" {
				if recovered != failure {
					t.Fatalf("panic changed: %v", recovered)
				}
			} else if recovered != nil {
				t.Fatalf("unexpected panic: %v", recovered)
			} else if outcome == "success" {
				if got != nil {
					t.Fatal(got)
				}
			} else if !errors.Is(got, failure) {
				t.Fatalf("error identity lost: %v", got)
			}
			spans := recorder.Ended()
			if len(spans) != 1 {
				t.Fatalf("ended spans=%d", len(spans))
			}
			span := spans[0]
			if span.Name() != "sql.transaction" || span.Parent().SpanID() != parent.SpanContext().SpanID() {
				t.Fatal("transaction parent/name incorrect")
			}
			if outcome == "success" {
				if span.Status().Code == codes.Error {
					t.Fatal("successful transaction reported as error")
				}
			} else if span.Status().Code != codes.Error {
				t.Fatalf("failed %s transaction status=%v", outcome, span.Status())
			}
			if !span.EndTime().After(span.StartTime()) {
				t.Fatal("invalid transaction timing")
			}
			for _, event := range span.Events() {
				for _, attribute := range event.Attributes {
					if strings.Contains(attribute.Value.AsString(), privateFailure) {
						t.Fatal("transaction span leaked private callback error text")
					}
				}
			}
			assertMock(t, db, mock)
		})
	}
}
