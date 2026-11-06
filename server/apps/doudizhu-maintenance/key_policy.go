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

// Safe until journal HMAC re-authentication and global new-only verification
// are implemented: neither the maintenance CLI nor an operator should mistake
// seed/checkpoint re-encryption for complete protection-root migration.
func checkPartialRotationAllowed(inventory protectionInventory) error {
	if inventory.actions != 0 {
		return fmt.Errorf("protected root rotation blocked: %d existing journal actions are not reauthenticated by this command; keep the old root, do not retire it (C09/C10 pending)", inventory.actions)
	}
	return nil
}
