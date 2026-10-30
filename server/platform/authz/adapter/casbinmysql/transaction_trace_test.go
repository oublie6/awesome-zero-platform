package casbinmysql

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/casbin/casbin/v2/model"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// Check SQL expectations synchronously at span end, not merely after the
// operation returns, so ending before rollback is a detectable regression.
type cleanupCheckingRecorder struct {
	*tracetest.SpanRecorder
	check func()
}

func (r *cleanupCheckingRecorder) OnEnd(span sdktrace.ReadOnlySpan) {
	if span.Name() == "sql.transaction" {
		r.check()
	}
	r.SpanRecorder.OnEnd(span)
}

func TestPolicyMutationFailureSpanIncludesRollback(t *testing.T) {
	for _, stage := range []string{"begin", "lock", "read", "transform"} {
		t.Run(stage, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			failure := errors.New("policy mutation failed")
			begin := mock.ExpectBegin()
			if stage == "begin" {
				begin.WillReturnError(failure)
			} else {
				lock := mock.ExpectQuery("SELECT version FROM authorization_policy_state.*FOR UPDATE").WithArgs(globalPolicyKey)
				if stage == "lock" {
					lock.WillReturnError(failure)
				} else {
					lock.WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(1))
					read := mock.ExpectQuery("SELECT ptype, v0, v1, v2, v3, v4, v5 FROM authorization_casbin_rules")
					if stage == "read" {
						read.WillReturnError(failure)
					} else {
						read.WillReturnRows(sqlmock.NewRows([]string{"ptype", "v0", "v1", "v2", "v3", "v4", "v5"})).RowsWillBeClosed()
					}
				}
				mock.ExpectRollback()
			}
			recorder := tracetest.NewSpanRecorder()
			processor := &cleanupCheckingRecorder{SpanRecorder: recorder, check: func() {
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("transaction span ended before cleanup: %v", err)
				}
			}}
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(processor))
			defer provider.Shutdown(context.Background())
			ctx, parent := provider.Tracer("test").Start(context.Background(), "request")
			defer parent.End()
			engine := &Engine{db: db, clusterEnabled: true}
			changed, got := engine.mutateRules(ctx, func([]authz.RawRule) ([]authz.RawRule, error) { return nil, failure })
			if changed || !errors.Is(got, failure) {
				t.Fatalf("mutation result changed=%v error=%v", changed, got)
			}
			spans := recorder.Ended()
			if len(spans) != 1 || spans[0].Name() != "sql.transaction" || spans[0].Status().Code != codes.Error || spans[0].Parent().SpanID() != parent.SpanContext().SpanID() {
				t.Fatal("policy transaction missing failed request-child span")
			}
			// All transaction cleanup must already have happened when the span ends.
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
			mock.ExpectClose()
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestContextlessCasbinSaveTransactionHasIndependentOutcomeSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer provider.Shutdown(context.Background())
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	defer otel.SetTracerProvider(previous)
	for _, failed := range []bool{false, true} {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		mock.ExpectBegin()
		deleted := mock.ExpectExec("DELETE FROM authorization_casbin_rules")
		failure := errors.New("save policy failed")
		if failed {
			deleted.WillReturnError(failure)
			mock.ExpectRollback()
		} else {
			deleted.WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()
		}
		adapter := &mysqlAdapter{db: db}
		got := adapter.SavePolicy(model.Model{})
		if failed && !errors.Is(got, failure) || !failed && got != nil {
			t.Fatalf("save outcome error=%v", got)
		}
		spans := recorder.Ended()
		if len(spans) == 0 {
			t.Fatal("contextless transaction span missing")
		}
		span := spans[len(spans)-1]
		if span.Name() != "sql.transaction" || span.Parent().IsValid() || (span.Status().Code == codes.Error) != failed {
			t.Fatal("contextless transaction outcome/root incorrect")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		mock.ExpectClose()
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if len(recorder.Ended()) != 2 {
		t.Fatal("unexpected transaction span count")
	}
}
