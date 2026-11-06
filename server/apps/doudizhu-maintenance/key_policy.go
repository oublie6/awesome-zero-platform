package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

// protectionInventory is metadata only. It never reads, prints or exports
// protection roots, plaintext seeds, checkpoints or action payloads.
type protectionInventory struct {
	actions       int64
	authenticated int64
	legacy        int64
	seeds         int64
	checkpoints   int64
}

func inspectProtectionInventory(ctx context.Context, db *sql.DB, hand domain.HandID) (protectionInventory, error) {
	if db == nil {
		return protectionInventory{}, fmt.Errorf("protection inventory requires mysql")
	}
	var result protectionInventory
	condition, args := "", []any(nil)
	if hand != "" {
		condition = " WHERE hand_id=?"
		args = append(args, hand)
	}
	journal := `SELECT COUNT(*),
COALESCE(SUM(CASE WHEN integrity_version=2 THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN integrity_version=1 THEN 1 ELSE 0 END),0)
FROM doudizhu_live_actions` + condition
	if err := db.QueryRowContext(ctx, journal, args...).Scan(
		&result.actions, &result.authenticated, &result.legacy,
	); err != nil {
		return protectionInventory{}, fmt.Errorf("inventory live journal metadata: %w", err)
	}
	for _, item := range []struct {
		table string
		count *int64
	}{
		{table: "doudizhu_protected_server_seeds", count: &result.seeds},
		{table: "doudizhu_live_checkpoints", count: &result.checkpoints},
	} {
		// The table names are compile-time constants, not user input.
		query := "SELECT COUNT(*) FROM " + item.table + condition
		if err := db.QueryRowContext(ctx, query, args...).Scan(item.count); err != nil {
			return protectionInventory{}, fmt.Errorf("inventory %s metadata: %w", item.table, err)
		}
	}
	return result, nil
}

// validateMaintenanceMode rejects the old partial rewrap entry point even
// with zero journal actions. A non-locking inventory check cannot establish
// writer quiescence, and seed/checkpoint rewrap used separate transactions.
// A future migration must be a different, atomic and independently verified
// administrator workflow.
func validateMaintenanceMode(mode string) error {
	switch mode {
	case "key-audit", "prune":
		return nil
	case "rotate":
		return fmt.Errorf("partial protection-root rotation is disabled: unsafe with live writers or process interruption (R12-N01); no root migration or retirement is supported (C09/C10 pending)")
	default:
		return fmt.Errorf("invalid maintenance mode")
	}
}
