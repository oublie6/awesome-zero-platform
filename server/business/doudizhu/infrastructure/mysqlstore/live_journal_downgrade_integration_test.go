//go:build integration

package mysqlstore_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
)

// Round 5 regression: configured authenticated recovery must not trust a
// row-selected unkeyed legacy version, even when its legacy checksum matches.
// Only disposable test-owned MySQL rows are modified.
func TestAuthenticatedLiveJournalRejectsLegacyDowngradeInMySQL(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("MySQL integration disabled")
	}
	db := openIntegrationDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	id := domain.HandID(fmt.Sprintf("r5-journal-downgrade-%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		for _, table := range []string{"doudizhu_live_checkpoints", "doudizhu_live_actions",
			"doudizhu_live_writers", "doudizhu_protected_server_seeds"} {
			_, _ = db.Exec("DELETE FROM "+table+" WHERE hand_id=?", id)
		}
	})
	seed, genesis, snapshot := coordinatorCrashFixture(t, id)
	seedStore, err := mysqlstore.NewProtectedSeedStore(db, coordinatorCrashKey())
	if err != nil {
		t.Fatal(err)
	}
	if err := seedStore.Save(ctx, id, seed); err != nil {
		t.Fatal(err)
	}
	journal, err := mysqlstore.NewLiveJournal(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.ConfigureCheckpoints(coordinatorCrashKey()); err != nil {
		t.Fatal(err)
	}
	if err := journal.Create(ctx, genesis); err != nil {
		t.Fatal(err)
	}
	load := func(_ context.Context, target domain.HandID) (domain.HandSnapshot, error) {
		if target != id {
			return domain.HandSnapshot{}, application.ErrNotFound
		}
		return snapshot, nil
	}
	active := newCoordinatorCrashRuntime(t, db)
	if err := active.RecoverActive(ctx, load); err != nil {
		t.Fatal(err)
	}
	public, err := active.PublicView(ctx, id, snapshot.Seats[0].AccountID)
	if err != nil {
		t.Fatal(err)
	}
	var view livehand.PublicView
	if err := json.Unmarshal(public.Payload, &view); err != nil {
		t.Fatal(err)
	}
	bidder := snapshot.Seats[view.Bidding.FirstBidder-1].AccountID
	const originalID = "r5-valid-sequenced-bid"
	const alteredID = "r5-relabelled-legacy-bid"
	originalResult, err := active.BidCommittedWithSequence(ctx, id, bidder,
		originalID, public.Version, bidding.ScoreThree, 91)
	if err != nil || originalResult.Version != 2 {
		t.Fatalf("authenticated bid did not commit: %+v %v", originalResult, err)
	}
	actions, err := journal.Read(ctx, id)
	if err != nil || len(actions) != 1 || actions[0].IntegrityVersion != 2 || actions[0].ClientSeq != 91 {
		t.Fatalf("expected authenticated journal row: %+v %v", actions, err)
	}
	original := actions[0]
	var originalTag []byte
	if err := db.QueryRowContext(ctx, "SELECT integrity_tag FROM doudizhu_live_actions WHERE hand_id=? AND state_version=2", id).Scan(&originalTag); err != nil {
		t.Fatal(err)
	}
	legacy := original
	legacy.ClientSeq = 0
	legacy.CommandID = alteredID
	legacy.IntegrityVersion = 1
	legacyTag := calculateLegacyDowngradeDigest(t, legacy)
	if _, err := db.ExecContext(ctx, `UPDATE doudizhu_live_actions
SET command_id=?,client_sequence=0,integrity_version=1,integrity_tag=?
WHERE hand_id=? AND state_version=2`, alteredID, legacyTag[:], id); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.Read(ctx, id); !errors.Is(err, mysqlstore.ErrLiveReplayIntegrity) {
		t.Fatalf("authenticated read accepted downgraded legacy identity: %v", err)
	}
	cold := newCoordinatorCrashRuntime(t, db)
	if err := cold.RecoverActive(ctx, load); !errors.Is(err, mysqlstore.ErrLiveReplayIntegrity) {
		t.Fatalf("recovery accepted untrusted legacy identity: %v", err)
	}
	if _, err := cold.PublicView(ctx, id, bidder); err == nil {
		t.Fatal("failed recovery published an active game view")
	}
	// A valid authenticated record must remain recoverable, with its original
	// command identity and client sequence, once the isolated fixture is restored.
	if _, err := db.ExecContext(ctx, `UPDATE doudizhu_live_actions
SET command_id=?,client_sequence=91,integrity_version=2,integrity_tag=?
WHERE hand_id=? AND state_version=2`, originalID, originalTag, id); err != nil {
		t.Fatal(err)
	}
	restored := newCoordinatorCrashRuntime(t, db)
	if err := restored.RecoverActive(ctx, load); err != nil {
		t.Fatalf("original authenticated history no longer recovers: %v", err)
	}
	replayed, err := restored.BidCommittedWithSequence(ctx, id, bidder,
		originalID, public.Version, bidding.ScoreThree, 91)
	if err != nil || !replayed.Historical || replayed.Version != originalResult.Version ||
		!bytes.Equal(replayed.Payload, originalResult.Payload) {
		t.Fatalf("original receipt changed after recovery: %+v %v", replayed, err)
	}
}

func calculateLegacyDowngradeDigest(t *testing.T, action mysqlstore.LiveAction) [32]byte {
	t.Helper()
	var payload any
	decoder := json.NewDecoder(bytes.NewReader(action.Payload))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := json.Marshal(struct {
		Domain          string
		HandID          domain.HandID
		Version         uint64
		Epoch           uint64
		CommandID       string
		Actor           domain.AccountID
		ActorPosition   uint8
		ExpectedVersion uint64
		ClientSeq       uint64
		Payload         json.RawMessage
		ResultDigest    [32]byte
	}{
		Domain: "doudizhu/live-action-integrity/v2:",
		HandID: action.HandID, Version: action.Version, Epoch: action.Epoch,
		CommandID: action.CommandID, Actor: action.Actor,
		ActorPosition: action.ActorPosition, ExpectedVersion: action.ExpectedVersion,
		ClientSeq: action.ClientSeq, Payload: canonical, ResultDigest: action.ResultDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(binding)
}
