package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Addr:             "127.0.0.1:3306",
		Database:         "awesome_zero_platform",
		User:             "app_local",
		Password:         "dev-only-password",
		Charset:          "utf8mb4",
		ParseTime:        true,
		Location:         "UTC",
		TimeZone:         "+00:00",
		Timeout:          time.Second,
		MaxOpenConns:     4,
		MaxIdleConns:     2,
		ConnMaxLifetime:  time.Minute,
		StartupTimeout:   time.Second,
		ReadinessTimeout: time.Second,
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

		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		resource := &Resource{
			beginTxFn: db.BeginTx,
		}

		err := resource.WithinTransaction(context.Background(), func(context.Context, *sql.Tx) error {
			return nil
		})
		if err != nil {
			t.Fatalf("WithinTransaction() error = %v", err)
		}
		assertMock(t, db, mock)
	})

	t.Run("rollback on error", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		resource := &Resource{
			beginTxFn: db.BeginTx,
		}

		err := resource.WithinTransaction(context.Background(), func(context.Context, *sql.Tx) error {
			return errors.New("boom")
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		assertMock(t, db, mock)
	})

	t.Run("propagates context to callback and transaction begin", func(t *testing.T) {
		t.Parallel()

		ctxKey := struct{}{}
		ctx := context.WithValue(context.Background(), ctxKey, "context-value")
		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		resource := &Resource{
			beginTxFn: func(beginCtx context.Context, options *sql.TxOptions) (*sql.Tx, error) {
				if got := beginCtx.Value(ctxKey); got != "context-value" {
					t.Fatalf("begin context value = %v, want context-value", got)
				}
				return db.BeginTx(beginCtx, options)
			},
		}

		err := resource.WithinTransaction(ctx, func(callbackCtx context.Context, tx *sql.Tx) error {
			if got := callbackCtx.Value(ctxKey); got != "context-value" {
				t.Fatalf("callback context value = %v, want context-value", got)
			}
			if tx == nil {
				t.Fatal("callback received nil transaction")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("WithinTransaction() error = %v", err)
		}
		assertMock(t, db, mock)
	})

	t.Run("preserves useful rollback errors", func(t *testing.T) {
		t.Parallel()

		callbackErr := errors.New("callback failed")
		rollbackErr := errors.New("rollback failed")
		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback().WillReturnError(rollbackErr)

		resource := &Resource{
			beginTxFn: db.BeginTx,
		}

		err := resource.WithinTransaction(context.Background(), func(context.Context, *sql.Tx) error {
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
		assertMock(t, db, mock)
	})

	t.Run("wraps begin error", func(t *testing.T) {
		t.Parallel()

		beginErr := errors.New("begin failed")
		resource := &Resource{
			beginTxFn: func(context.Context, *sql.TxOptions) (*sql.Tx, error) {
				return nil, beginErr
			},
		}

		err := resource.WithinTransaction(context.Background(), func(context.Context, *sql.Tx) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected begin error, got nil")
		}
		if !errors.Is(err, beginErr) {
			t.Fatalf("wrapped error does not include begin error: %v", err)
		}
	})

	t.Run("wraps commit error", func(t *testing.T) {
		t.Parallel()

		commitErr := errors.New("commit failed")
		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit().WillReturnError(commitErr)

		resource := &Resource{
			beginTxFn: db.BeginTx,
		}

		err := resource.WithinTransaction(context.Background(), func(context.Context, *sql.Tx) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected commit error, got nil")
		}
		if !errors.Is(err, commitErr) {
			t.Fatalf("wrapped error does not include commit error: %v", err)
		}
		assertMock(t, db, mock)
	})

	t.Run("rollback on panic", func(t *testing.T) {
		t.Parallel()

		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		resource := &Resource{
			beginTxFn: db.BeginTx,
		}

		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
			assertMock(t, db, mock)
		}()

		_ = resource.WithinTransaction(context.Background(), func(context.Context, *sql.Tx) error {
			panic("panic")
		})
	})
}

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}

	return db, mock
}

func assertMock(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
	t.Helper()

	mock.ExpectClose()
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
