package identity

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

type recordingAccountStateQueryer struct {
	tx   *sql.Tx
	seen context.Context
}

func (r *recordingAccountStateQueryer) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	r.seen = ctx
	return r.tx.QueryContext(ctx, query, args...)
}

func TestActiveAccountGuardPreservesTransactionLocksAndFullResultErrors(t *testing.T) {
	for _, scenario := range []string{"active", "disabled", "missing", "query-error", "scan-error", "row-error-after-active"} {
		t.Run(scenario, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			tx, err := db.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			failure := errors.New("account state query failure")
			rows := sqlmock.NewRows([]string{"account_id", "status"})
			switch scenario {
			case "active":
				rows.AddRow("a", "active").AddRow("b", "disabled")
			case "disabled":
				rows.AddRow("a", "disabled").AddRow("b", "disabled")
			case "scan-error":
				rows.AddRow(nil, nil)
			case "row-error-after-active":
				rows.AddRow("a", "active").AddRow("b", "disabled").RowError(1, failure)
			}
			expectation := mock.ExpectQuery("SELECT account_id, status.*FROM identity_accounts.*FOR UPDATE").WithArgs("a", "b")
			if scenario == "query-error" {
				expectation.WillReturnError(failure)
			} else {
				expectation.WillReturnRows(rows).RowsWillBeClosed()
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reader := &recordingAccountStateQueryer{tx: tx}
			active, got := AnyActiveAccountForUpdate(ctx, reader, []string{"a", "b"})
			if reader.seen != ctx {
				t.Fatal("account guard lost caller transaction context")
			}
			switch scenario {
			case "query-error", "row-error-after-active":
				if !errors.Is(got, failure) {
					t.Fatalf("query error identity lost: %v", got)
				}
				if active {
					t.Fatal("failed result must not authorize an active account")
				}
			case "scan-error":
				if got == nil || active {
					t.Fatal("invalid account state row accepted")
				}
			default:
				if got != nil || active != (scenario == "active") {
					t.Fatalf("active=%v error=%v", active, got)
				}
			}
			// Before rollback, require the identity reader to have closed its cursor.
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
			mock.ExpectRollback()
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
			mock.ExpectClose()
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestActiveAccountGuardEmptyInputDoesNotQuery(t *testing.T) {
	active, err := AnyActiveAccountForUpdate(context.Background(), nil, nil)
	if active || err != nil {
		t.Fatalf("empty account set: active=%v error=%v", active, err)
	}
	if _, err := AnyActiveAccountForUpdate(context.Background(), nil, []string{"a"}); err == nil {
		t.Fatal("missing account state queryer accepted")
	}
}
