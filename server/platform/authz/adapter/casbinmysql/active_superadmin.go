package casbinmysql

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
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

	active, err := identity.AnyActiveAccountForUpdate(ctx, queryer, accountIDs)
	if err != nil {
		return fmt.Errorf("validate active platform super administrator: %w", err)
	}

	if !active {
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
