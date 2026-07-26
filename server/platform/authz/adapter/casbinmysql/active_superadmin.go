package casbinmysql

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
)

// validateActiveSuperAdminSafety ensures that policy writes cannot preserve a
// syntactically valid super-admin membership that points only at disabled or
// missing accounts. When queryer is a transaction, FOR UPDATE also coordinates
// the policy mutation with concurrent account status changes.
func validateActiveSuperAdminSafety(ctx context.Context, queryer policyQueryer, rules []authz.RawRule) error {
	accountIDs := directSuperAdminAccountIDs(rules)
	if len(accountIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(accountIDs))
	args := make([]any, len(accountIDs))
	for index, accountID := range accountIDs {
		placeholders[index] = "?"
		args[index] = accountID
	}
	rows, err := queryer.QueryContext(ctx, `SELECT account_id, status
		FROM identity_accounts
		WHERE account_id IN (`+strings.Join(placeholders, ",")+`)
		FOR UPDATE`, args...)
	if err != nil {
		return fmt.Errorf("validate active platform super administrator: %w", err)
	}
	defer rows.Close()

	active := 0
	for rows.Next() {
		var accountID, status string
		if err := rows.Scan(&accountID, &status); err != nil {
			return fmt.Errorf("scan platform super administrator status: %w", err)
		}
		if status == "active" {
			active++
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read platform super administrator status: %w", err)
	}
	if active == 0 {
		return fmt.Errorf("the authorization policy must retain at least one active platform super administrator")
	}
	return nil
}

func directSuperAdminAccountIDs(rules []authz.RawRule) []string {
	seen := make(map[string]struct{})
	for _, rule := range rules {
		values := trimmedValues(rule.Values)
		if strings.TrimSpace(rule.PType) != "g" || len(values) != 2 || values[1] != protectedSuperAdminRole {
			continue
		}
		if values[0] == "" {
			continue
		}
		seen[values[0]] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for accountID := range seen {
		result = append(result, accountID)
	}
	sort.Strings(result)
	return result
}
