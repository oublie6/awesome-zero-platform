package identity

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// AccountStateQueryer is the narrow infrastructure port for an in-process
// transaction that coordinates account state with another module's invariant.
// The caller supplies its existing transaction; identity owns the SQL and
// interpretation of account status. This is explicitly a monolith transaction
// contract, not an RPC contract: extraction requires redesigning the atomic
// invariant rather than replacing this call with a remote lookup.
type AccountStateQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// AnyActiveAccountForUpdate locks all matching accounts in the supplied
// transaction and reports whether at least one is active. Native cursor access
// preserves the existing FOR UPDATE and full-result/error/lock semantics.
// Nontransactional callers retain their existing statement-scoped locking.
func AnyActiveAccountForUpdate(ctx context.Context, queryer AccountStateQueryer, accountIDs []string) (bool, error) {
	if len(accountIDs) == 0 {
		return false, nil
	}
	if queryer == nil {
		return false, fmt.Errorf("account state queryer is required")
	}
	placeholders := make([]string, len(accountIDs))
	args := make([]any, len(accountIDs))
	for i, id := range accountIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := queryer.QueryContext(ctx, `SELECT account_id, status
  FROM identity_accounts
  WHERE account_id IN (`+strings.Join(placeholders, ",")+`)
  FOR UPDATE`, args...)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	active := false
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			return false, err
		}
		if status == string(StatusActive) {
			active = true
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return active, nil
}
