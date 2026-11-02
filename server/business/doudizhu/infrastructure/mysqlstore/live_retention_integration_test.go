//go:build integration

package mysqlstore_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/randomizedsetup"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore/infrastructure/mysqlarchive"
)

type retentionClock struct{ at time.Time }

func (c retentionClock) Now() time.Time { return c.at }

type retentionProof struct {
	record   gamecore.FinalRecord
	archived time.Time
	err      error
}

func (r retentionProof) LoadFinalRecord(context.Context, gamecore.InstanceID) (gamecore.FinalRecord, time.Time, error) {
	return r.record, r.archived, r.err
}

func TestLiveArchivePruningRequiresVerifiedFinalAndThirtyDayRetention(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("requires real MySQL")
	}
	db := openIntegrationDB(t)
	ctx := context.Background()
	id := domain.HandID(fmt.Sprintf("prune-final-%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		for _, table := range []string{"doudizhu_live_checkpoints", "doudizhu_live_actions",
			"doudizhu_live_writers", "doudizhu_protected_server_seeds", "game_final_records", "doudizhu_hands"} {
			column := "hand_id"
			if table == "game_final_records" {
				column = "instance_id"
			}
			_, _ = db.Exec("DELETE FROM "+table+" WHERE "+column+"=?", id)
		}
	})
	seed, genesis, _ := coordinatorCrashFixture(t, id)
	seeds, err := mysqlstore.NewProtectedSeedStore(db, coordinatorCrashKey())
	if err != nil {
		t.Fatal(err)
	}
	if err := seeds.Save(ctx, id, seed); err != nil {
		t.Fatal(err)
	}
	journal, err := mysqlstore.NewLiveJournal(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Create(ctx, genesis); err != nil {
		t.Fatal(err)
	}
	owner, _, _, err := journal.Acquire(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	command := gamecore.Command{ActorPosition: 1, ExpectedVersion: 1,
		Payload: []byte(`{"v":"doudizhu-live-bid-command-v1","score":3}`)}
	result := gamecore.CommandOutcome{Version: 2, Payload: []byte(`{"stateVersion":2}`)}
	if err := journal.Append(ctx, id, owner, "prune-final-action", "crash-player-1", command, result); err != nil {
		t.Fatal(err)
	}
	if err := journal.PruneArchivedHand(ctx, id, 30*24*time.Hour, retentionProof{err: sql.ErrNoRows}); !errors.Is(err, mysqlstore.ErrLivePruneNotEligible) {
		t.Fatalf("live hand pruned: %v", err)
	}
	if err := journal.PruneArchivedHand(ctx, id, 24*time.Hour, retentionProof{}); !errors.Is(err, mysqlstore.ErrLivePruneNotEligible) {
		t.Fatalf("retention below minimum permitted: %v", err)
	}
	// Create a completed authoritative domain projection and a valid archive;
	// no faked/synthetic final digest is accepted by the pruning implementation.
	if _, err := db.ExecContext(ctx, `INSERT INTO doudizhu_hands(
hand_id,room_id,phase,reveal_key_id,reveal_public_key_sha256,aggregate_version,
snapshot_json,created_at,updated_at) VALUES(?,?,'COMPLETED',?,REPEAT(0x00,32),1,'{}',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`,
		id, "prune-room", "test-key"); err != nil {
		t.Fatal(err)
	}
	desc := randomizedsetup.Descriptor()
	final, err := gamecore.NewFinalRecord(gamecore.InstanceID(id), desc, gamecore.FinalStatusCompleted, 2, []byte(`{"settled":true}`))
	if err != nil {
		t.Fatal(err)
	}
	archive, err := mysqlarchive.New(db, retentionClock{at: time.Now().UTC().Add(-45 * 24 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err := archive.Archive(final); err != nil {
		t.Fatal(err)
	}
	if err := journal.PruneArchivedHand(ctx, id, 30*24*time.Hour, retentionProof{record: final, archived: time.Now()}); !errors.Is(err, mysqlstore.ErrLivePruneNotEligible) {
		t.Fatal("young final pruned:", err)
	}
	if err := journal.PruneArchivedHand(ctx, id, 30*24*time.Hour, retentionProof{record: gamecore.FinalRecord{}, archived: time.Now().Add(-45 * 24 * time.Hour)}); !errors.Is(err, mysqlstore.ErrLiveReplayIntegrity) {
		t.Fatal("forged archive pruned:", err)
	}
	if err := journal.PruneArchivedHand(ctx, id, 30*24*time.Hour, archive); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM doudizhu_live_writers WHERE hand_id=?`, id).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("prune left writer behind: %d %v", remaining, err)
	}
	stored, _, err := archive.LoadFinalRecord(ctx, gamecore.InstanceID(id))
	if err != nil || !bytes.Equal(stored.Payload(), final.Payload()) || stored.Digest() != final.Digest() {
		t.Fatalf("final evidence altered during prune: %v", err)
	}
}
