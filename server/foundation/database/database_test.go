package database

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Host:              "127.0.0.1",
		Port:              5432,
		Database:          "awesome_zero_platform",
		User:              "app_local",
		Password:          "dev-only-password",
		SSLMode:           "disable",
		MaxConns:          4,
		MinConns:          0,
		ConnectTimeout:    time.Second,
		StartupTimeout:    time.Second,
		ReadinessTimeout:  time.Second,
		HealthCheckPeriod: time.Second,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestCloseIdempotent(t *testing.T) {
	t.Parallel()

	calls := 0
	resource := &Resource{
		closeFn: func() error {
			calls++
			return nil
		},
	}

	if err := resource.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := resource.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("close calls = %d, want 1", calls)
	}
}

func TestWithinTransactionCommitRollbackAndPanic(t *testing.T) {
	t.Parallel()

	t.Run("commit on success", func(t *testing.T) {
		t.Parallel()

		tx := &stubTx{}
		resource := &Resource{
			beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
				return tx, nil
			},
		}

		err := resource.WithinTransaction(context.Background(), func(context.Context, pgx.Tx) error {
			return nil
		})
		if err != nil {
			t.Fatalf("WithinTransaction() error = %v", err)
		}
		if !tx.committed || tx.rolledBack {
			t.Fatalf("unexpected commit/rollback state: committed=%v rolledBack=%v", tx.committed, tx.rolledBack)
		}
	})

	t.Run("rollback on error", func(t *testing.T) {
		t.Parallel()

		tx := &stubTx{}
		resource := &Resource{
			beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
				return tx, nil
			},
		}

		err := resource.WithinTransaction(context.Background(), func(context.Context, pgx.Tx) error {
			return errors.New("boom")
		})
		if err == nil || !tx.rolledBack || tx.committed {
			t.Fatalf("unexpected error or transaction state: err=%v committed=%v rolledBack=%v", err, tx.committed, tx.rolledBack)
		}
	})

	t.Run("propagates context to callback and transaction begin", func(t *testing.T) {
		t.Parallel()

		ctxKey := struct{}{}
		ctx := context.WithValue(context.Background(), ctxKey, "context-value")
		tx := &stubTx{}
		resource := &Resource{
			beginTxFn: func(beginCtx context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
				if got := beginCtx.Value(ctxKey); got != "context-value" {
					t.Fatalf("begin context value = %v, want context-value", got)
				}
				return tx, nil
			},
		}

		err := resource.WithinTransaction(ctx, func(callbackCtx context.Context, callbackTx pgx.Tx) error {
			if got := callbackCtx.Value(ctxKey); got != "context-value" {
				t.Fatalf("callback context value = %v, want context-value", got)
			}
			if callbackTx != tx {
				t.Fatal("callback received unexpected transaction handle")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("WithinTransaction() error = %v", err)
		}
	})

	t.Run("preserves useful rollback errors", func(t *testing.T) {
		t.Parallel()

		callbackErr := errors.New("callback failed")
		rollbackErr := errors.New("rollback failed")
		tx := &stubTx{rollbackErr: rollbackErr}
		resource := &Resource{
			beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
				return tx, nil
			},
		}

		err := resource.WithinTransaction(context.Background(), func(context.Context, pgx.Tx) error {
			return callbackErr
		})
		if err == nil {
			t.Fatal("expected joined error, got nil")
		}
		if !errors.Is(err, callbackErr) {
			t.Fatalf("joined error does not include callback error: %v", err)
		}
		if !errors.Is(err, rollbackErr) {
			t.Fatalf("joined error does not include rollback error: %v", err)
		}
	})

	t.Run("wraps begin error", func(t *testing.T) {
		t.Parallel()

		beginErr := errors.New("begin failed")
		resource := &Resource{
			beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
				return nil, beginErr
			},
		}

		err := resource.WithinTransaction(context.Background(), func(context.Context, pgx.Tx) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected begin error, got nil")
		}
		if !errors.Is(err, beginErr) {
			t.Fatalf("wrapped error does not include begin error: %v", err)
		}
		if got := err.Error(); got != "begin transaction: begin failed" {
			t.Fatalf("error message = %q, want %q", got, "begin transaction: begin failed")
		}
	})

	t.Run("wraps commit error", func(t *testing.T) {
		t.Parallel()

		commitErr := errors.New("commit failed")
		tx := &stubTx{commitErr: commitErr}
		resource := &Resource{
			beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
				return tx, nil
			},
		}

		err := resource.WithinTransaction(context.Background(), func(context.Context, pgx.Tx) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected commit error, got nil")
		}
		if !errors.Is(err, commitErr) {
			t.Fatalf("wrapped error does not include commit error: %v", err)
		}
		if got := err.Error(); got != "commit transaction: commit failed" {
			t.Fatalf("error message = %q, want %q", got, "commit transaction: commit failed")
		}
	})

	t.Run("rollback on panic", func(t *testing.T) {
		t.Parallel()

		tx := &stubTx{}
		resource := &Resource{
			beginTxFn: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
				return tx, nil
			},
		}

		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
			if !tx.rolledBack {
				t.Fatal("expected rollback on panic")
			}
		}()

		_ = resource.WithinTransaction(context.Background(), func(context.Context, pgx.Tx) error {
			panic("panic")
		})
	})
}

type stubTx struct {
	committed   bool
	rolledBack  bool
	commitErr   error
	rollbackErr error
}

func (s *stubTx) Begin(context.Context) (pgx.Tx, error) { return nil, errors.New("not implemented") }
func (s *stubTx) Commit(context.Context) error {
	s.committed = true
	return s.commitErr
}
func (s *stubTx) Rollback(context.Context) error {
	s.rolledBack = true
	return s.rollbackErr
}
func (s *stubTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (s *stubTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (s *stubTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (s *stubTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (s *stubTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("not implemented")
}
func (s *stubTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (s *stubTx) QueryRow(context.Context, string, ...any) pgx.Row { return nil }
func (s *stubTx) Conn() *pgx.Conn                                  { return nil }
func (s *stubTx) String() string {
	return fmt.Sprintf("stubTx(committed=%v, rolledBack=%v)", s.committed, s.rolledBack)
}
