// Doudizhu maintenance is an explicitly authorized offline operator action.
// It never starts a game, issues client commands or changes final evidence.
package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
)

var safeHand = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func previousRoots() (string, map[string][]byte, error) {
	current := os.Getenv("APP_DOUDIZHU_CONTRIBUTION_KEY_ID")
	active, err := hex.DecodeString(os.Getenv("APP_DOUDIZHU_CONTRIBUTION_KEY_HEX"))
	if err != nil || current == "" || len(active) != 32 {
		return "", nil, fmt.Errorf("active 256-bit Doudizhu root and key ID are required")
	}
	result := map[string][]byte{current: active}
	if raw := os.Getenv("APP_DOUDIZHU_PREVIOUS_KEYS_JSON"); raw != "" {
		var previous map[string]string
		if err := json.Unmarshal([]byte(raw), &previous); err != nil {
			return "", nil, fmt.Errorf("malformed previous-key map")
		}
		if len(previous) > 7 {
			return "", nil, fmt.Errorf("too many previous keys")
		}
		for id, encoded := range previous {
			if id == "" || len(id) > 128 || id == current {
				return "", nil, fmt.Errorf("invalid previous key identity")
			}
			key, err := hex.DecodeString(encoded)
			if err != nil || len(key) != 32 {
				return "", nil, fmt.Errorf("invalid previous key length")
			}
			result[id] = key
		}
	}
	return current, result, nil
}

func run() error {
	mode := flag.String("mode", "", "rotate or prune")
	hand := flag.String("hand", "", "explicit hand ID")
	epoch := flag.Uint64("epoch", 0, "currently fenced owner epoch; required for rotate")
	retention := flag.Int("retention-days", 30, "minimum archived history age before pruning")
	apply := flag.Bool("execute", false, "required to perform a mutation; without it this is a dry run")
	flag.Parse()
	if !safeHand.MatchString(*hand) || (*mode != "rotate" && *mode != "prune") {
		return fmt.Errorf("invalid mode or hand ID")
	}
	if *mode == "rotate" && *epoch == 0 {
		return fmt.Errorf("checkpoint rotation requires a positive fenced owner epoch")
	}
	if *mode == "prune" && (*retention < 30 || *retention > 3650) {
		return fmt.Errorf("history retention must be between 30 and 3650 days")
	}
	if !*apply {
		fmt.Printf("DRY_RUN: mode=%s hand=%s; rerun with --execute after verifying the archive and current owner\n", *mode, *hand)
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
	case "prune":
		return journal.PruneArchivedHand(ctx, id, time.Duration(*retention)*24*time.Hour)
	case "rotate":
		current, roots, err := previousRoots()
		if err != nil {
			return err
		}
		defer func() {
			for _, key := range roots {
				clear(key)
			}
		}()
		seed, err := mysqlstore.NewProtectedSeedStoreKeyring(db, current, roots)
		if err != nil {
			return err
		}
		if err := journal.ConfigureCheckpointKeyring(current, roots); err != nil {
			return err
		}
		// Independent transactions are deliberately retryable: old roots must
		// remain configured until both rewrap operations have succeeded.
		if err := seed.ReencryptSeed(ctx, id, db); err != nil {
			return fmt.Errorf("rotate protected server seed: %w", err)
		}
		if err := journal.ReencryptCheckpoint(ctx, id, *epoch); err != nil {
			return fmt.Errorf("rotate fenced checkpoint: %w", err)
		}
		return nil
	}
	return fmt.Errorf("unrecognized maintenance mode")
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Doudizhu maintenance failed:", err)
		os.Exit(1)
	}
}
