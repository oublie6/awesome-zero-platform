package database

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestWithAdvisoryLockIsReentrantInCallbackContext(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT GET_LOCK\(\?, \?\)`).
		WithArgs("shared-lock", int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(1))
	mock.ExpectQuery(`SELECT RELEASE_LOCK\(\?\)`).
		WithArgs("shared-lock").
		WillReturnRows(sqlmock.NewRows([]string{"released"}).AddRow(1))

	calls := 0
	err = WithAdvisoryLock(context.Background(), db, "shared-lock", 1500*time.Millisecond, func(lockCtx context.Context) error {
		calls++
		return WithAdvisoryLock(lockCtx, db, "shared-lock", time.Second, func(context.Context) error {
			calls++
			return nil
		})
	})
	if err != nil {
		t.Fatalf("WithAdvisoryLock() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("callback calls = %d, want 2", calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
