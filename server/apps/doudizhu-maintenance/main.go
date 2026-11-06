// Doudizhu maintenance is an explicitly authorized offline operator action.
// It never starts a game, issues client commands or changes final evidence.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore/infrastructure/mysqlarchive"
)

var safeHand = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type maintenanceClock struct{}

func (maintenanceClock) Now() time.Time { return time.Now().UTC() }

func run() error {
	mode := flag.String("mode", "", "key-audit or prune (rotation is disabled)")
	hand := flag.String("hand", "", "explicit hand ID")
	_ = flag.Uint64("epoch", 0, "legacy option: rotate is disabled")
	retention := flag.Int("retention-days", 30, "minimum archived history age before pruning")
	apply := flag.Bool("execute", false, "required to perform a mutation; without it this is a dry run")
	flag.Parse()
	// Reject unsafe historical rotate *before* opening a database, parsing
	// root secrets or making any mutation. An inventory check cannot fence
	// an arriving first Journal action (R12-N01).
	if err := validateMaintenanceMode(*mode); err != nil {
		return err
	}
	if *mode != "key-audit" && !safeHand.MatchString(*hand) {
		return fmt.Errorf("prune requires an explicit valid hand ID")
	}
	if *mode == "key-audit" && (*apply || (*hand != "" && !safeHand.MatchString(*hand))) {
		return fmt.Errorf("key-audit is read-only and accepts only an optional valid hand ID")
	}
	if *mode == "prune" && (*retention < 30 || *retention > 3650) {
		return fmt.Errorf("history retention must be between 30 and 3650 days")
	}
	if !*apply && *mode != "key-audit" {
		fmt.Printf("DRY_RUN: mode=%s hand=%s; no database checked; run --mode key-audit and obtain operator approval before --execute\n", *mode, *hand)
		return nil
	}
	dsn := os.Getenv("APP_MYSQL_DSN")
	if dsn == "" {
		return fmt.Errorf("APP_MYSQL_DSN required (credentials are not accepted on the command line)")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("connect maintenance database: %w", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database readiness: %w", err)
	}
	journal, err := mysqlstore.NewLiveJournal(db)
	if err != nil {
		return err
	}
	id := domain.HandID(*hand)
	switch *mode {
	case "key-audit":
		inventory, err := inspectProtectionInventory(ctx, db, id)
		if err != nil {
			return err
		}
		// These are metadata counts, not a proof that every record can be
		// recovered using the active root alone. Never print key material.
		fmt.Printf("KEY_AUDIT hand=%s journal_actions=%d authenticated_v2=%d legacy_v1=%d seeds=%d checkpoints=%d old_root_retirement=UNVERIFIED (C09/C10)\n",
			*hand, inventory.actions, inventory.authenticated, inventory.legacy, inventory.seeds, inventory.checkpoints)
		return nil
	case "prune":
		archive, err := mysqlarchive.New(db, maintenanceClock{})
		if err != nil {
			return err
		}
		return journal.PruneArchivedHand(ctx, id, time.Duration(*retention)*24*time.Hour, archive)

	}
	return fmt.Errorf("unrecognized maintenance mode")
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Doudizhu maintenance failed:", err)
		os.Exit(1)
	}
}
