//go:build integration

package mysqlstore_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
	"os"
)

func TestLiveJournalFenceContiguousLogAndProtectedSeedInMySQL(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("mysql integration disabled")
	}
	db := openIntegrationDB(t)
	ctx := context.Background()
	id := domain.HandID(fmt.Sprintf("live-journal-%d", time.Now().UnixNano()))
	defer func() {
		db.Exec("DELETE FROM doudizhu_live_checkpoints WHERE hand_id=?", id)
		db.Exec("DELETE FROM doudizhu_live_actions WHERE hand_id=?", id)
		db.Exec("DELETE FROM doudizhu_live_writers WHERE hand_id=?", id)
		db.Exec("DELETE FROM doudizhu_protected_server_seeds WHERE hand_id=?", id)
	}()
	seats := [3]domain.HandSeat{
		{Seat: 1, AccountID: "player-1"}, {Seat: 2, AccountID: "player-2"}, {Seat: 3, AccountID: "player-3"},
	}
	server := domain.ServerCommitment{1}
	hand, _, err := domain.NewHand(id, domain.RoomID("room-test"), seats, server, "test-key", domain.BeaconPlan{Provider: "test-provider", Round: "test-round"})
	if err != nil {
		t.Fatal(err)
	}
	for index, seat := range seats {
		var contribution domain.ContributionDigest
		contribution[0] = byte(index + 1)
		commitment := domain.ComputeClientCommitment(id, seat.Seat, contribution)
		if _, err := hand.SubmitCommit(seat.AccountID, commitment, hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	for index, seat := range seats {
		var contribution domain.ContributionDigest
		contribution[0] = byte(index + 1)
		if _, err := hand.SubmitReveal(seat.AccountID, contribution, fmt.Sprintf("record-%d", index), hand.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	beacon := domain.BeaconDigest{3}
	if _, err := hand.LockPublicBeacon(domain.BeaconValue{Provider: "test-provider", Round: "test-round", Digest: beacon, ProofRef: "proof-test"}, hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if hand.Snapshot().Phase != domain.HandDealing {
		t.Fatalf("genesis not dealing")
	}
	j, err := mysqlstore.NewLiveJournal(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Create(ctx, hand.Snapshot()); err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{0x5a}, 32)
	seeds, err := mysqlstore.NewProtectedSeedStore(db, key)
	if err != nil {
		t.Fatal(err)
	}
	var original carddeck.Seed
	for i := range original {
		original[i] = byte(i + 1)
	}
	if err := seeds.Save(ctx, id, original); err != nil {
		t.Fatal(err)
	}
	restored, err := mysqlstore.NewProtectedSeedStore(db, key)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := restored.Load(ctx, id)
	if err != nil || recovered != original {
		t.Fatalf("seed not restored: %v", err)
	}
	var stored []byte
	if err := db.QueryRow("SELECT ciphertext FROM doudizhu_protected_server_seeds WHERE hand_id=?", id).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stored, original[:]) {
		t.Fatal("unencrypted seed persisted")
	}
	epoch, version, genesis, err := j.Acquire(ctx, id)
	if err != nil || epoch != 2 || version != 1 || genesis.ID != id {
		t.Fatalf("takeover epoch=%d version=%d genesis=%s err=%v", epoch, version, genesis.ID, err)
	}
	action := gamecore.Command{ActorPosition: 1, ExpectedVersion: 1, Payload: []byte(`{"v":"doudizhu-live-bid-command-v1","score":1}`)}
	result := gamecore.CommandOutcome{Version: 2, Payload: []byte(`{"ok":true}`)}
	if err := j.Append(ctx, id, epoch, "act1", "player-1", action, result); err != nil {
		t.Fatal(err)
	}
	actions, err := j.Read(ctx, id)
	if err != nil || len(actions) != 1 || actions[0].Version != 2 || actions[0].CommandID != "act1" {
		t.Fatalf("log=%+v err=%v", actions, err)
	}
	// Replay must refuse tampered metadata, even when the recorded result
	// digest has not been changed. Restore each row before the next test.
	for _, mutation := range []struct {
		name     string
		column   string
		corrupt  any
		restored any
	}{
		{"zero owner epoch", "owner_epoch", uint64(0), epoch},
		{"empty command identity", "command_id", "", "act1"},
		{"empty actor", "actor_account_id", "", "player-1"},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			query := fmt.Sprintf("UPDATE doudizhu_live_actions SET %s=? WHERE hand_id=? AND state_version=2", mutation.column)
			if _, err := db.ExecContext(ctx, query, mutation.corrupt, id); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if _, err := db.ExecContext(ctx, query, mutation.restored, id); err != nil {
					t.Error(err)
				}
			}()
			if _, err := j.Read(ctx, id); !errors.Is(err, mysqlstore.ErrLiveReplayIntegrity) {
				t.Fatalf("tampered live journal metadata accepted: %v", err)
			}
		})
	}
	// Authenticated encrypted checkpoint stays recoverable across a new
	// journal instance, and never persists a private card/seed in cleartext.
	if err := j.ConfigureCheckpoints(key); err != nil {
		t.Fatal(err)
	}
	private := []byte(`{"v":"doudizhu-live-recovery-checkpoint-v1","private":"S3CRET_CARDS"}`)
	if err := j.SaveCheckpoint(ctx, id, epoch, 2, private); err != nil {
		t.Fatal(err)
	}
	loaded, err := j.LoadCheckpoint(ctx, id)
	if err != nil || loaded.Version != 2 || !bytes.Equal(loaded.Payload, private) {
		t.Fatalf("encrypted checkpoint failed round-trip: %+v %v", loaded, err)
	}
	var ciphertext []byte
	if err := db.QueryRowContext(ctx, "SELECT ciphertext FROM doudizhu_live_checkpoints WHERE hand_id=?", id).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, []byte("S3CRET_CARDS")) {
		t.Fatal("private checkpoint payload persisted in plaintext")
	}
	if _, err := db.ExecContext(ctx, "UPDATE doudizhu_live_checkpoints SET ciphertext=REPEAT(0x00,64) WHERE hand_id=?", id); err != nil {
		t.Fatal(err)
	}
	if _, err := j.LoadCheckpoint(ctx, id); !errors.Is(err, mysqlstore.ErrLiveReplayIntegrity) {
		t.Fatalf("tampered checkpoint accepted: %v", err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE doudizhu_live_checkpoints SET ciphertext=? WHERE hand_id=?", ciphertext, id); err != nil {
		t.Fatal(err)
	}
	if err := j.SaveCheckpoint(ctx, id, epoch, 3, private); !errors.Is(err, mysqlstore.ErrLiveWatermarkConflict) {
		t.Fatalf("future checkpoint accepted: %v", err)
	}
	newEpoch, watermark, _, err := j.Acquire(ctx, id)
	if err != nil || newEpoch != 3 || watermark != 2 {
		t.Fatalf("second owner epoch=%d watermark=%d err=%v", newEpoch, watermark, err)
	}
	next := gamecore.Command{ActorPosition: 2, ExpectedVersion: 2, Payload: []byte(`{"v":"doudizhu-live-pass-command-v1"}`)}
	if err := j.SaveCheckpoint(ctx, id, epoch, 2, private); !errors.Is(err, mysqlstore.ErrLiveWriterFenced) {
		t.Fatalf("fenced owner overwrote protected checkpoint: %v", err)
	}
	if err := j.Append(ctx, id, epoch, "stale", "player-2", next, gamecore.CommandOutcome{Version: 3, Payload: []byte(`{"ok":true}`)}); !errors.Is(err, mysqlstore.ErrLiveWriterFenced) {
		t.Fatalf("old owner not fenced: %v", err)
	}
	if err := j.Append(ctx, id, newEpoch, "gap", "player-2", gamecore.Command{ActorPosition: 2, ExpectedVersion: 4, Payload: next.Payload}, gamecore.CommandOutcome{Version: 5, Payload: []byte(`{"ok":true}`)}); !errors.Is(err, mysqlstore.ErrLiveWatermarkConflict) {
		t.Fatalf("version gap accepted: %v", err)
	}
	if _, err := db.Exec("UPDATE doudizhu_live_actions SET result_digest=REPEAT(0x00,32) WHERE hand_id=? AND state_version=2", id); err != nil {
		t.Fatal(err)
	}
	// The loader returns hashes, and recovery must validate them against
	// deterministic Go Apply output before exposing a hand.
	// The authenticated request identity now includes the outcome digest;
	// any field mutation fails closed before receipt reconstruction.
	if _, err := j.Read(ctx, id); !errors.Is(err, mysqlstore.ErrLiveReplayIntegrity) {
		t.Fatalf("tampered result digest was accepted: %v", err)
	}
	if _, err := db.Exec("UPDATE doudizhu_protected_server_seeds SET ciphertext=REPEAT(0x00,48) WHERE hand_id=?", id); err != nil {
		t.Fatal(err)
	}
	if _, err := restored.Load(ctx, id); err == nil {
		t.Fatal("tampered encrypted seed accepted")
	}
}
