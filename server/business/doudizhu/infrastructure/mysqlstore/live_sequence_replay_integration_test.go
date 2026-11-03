//go:build integration

package mysqlstore_test

import (
	"bytes"
	"context"
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
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/protection"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/textnormalization"
	"github.com/oublie6/awesome-zero-platform/server/business/gamecore"
)

// This reproduces the original R3-N03: a committed live bid, phase advancement,
// real MySQL takeover and a first post-restart retry with only clientSeq changed.
// It must fail before populating a hot Dispatcher replay cache with the forged
// identity, and the ORIGINAL sequence must remain retryable afterwards.
func TestRealMySQLClientSequenceBoundAcrossProcessRecovery(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("MySQL integration disabled")
	}
	db := openIntegrationDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	id := domain.HandID(fmt.Sprintf("live-clientseq-%d", time.Now().UnixNano()))
	t.Cleanup(func() {
		for _, table := range []string{"doudizhu_live_checkpoints", "doudizhu_live_actions",
			"doudizhu_live_writers", "doudizhu_protected_server_seeds", "doudizhu_hands"} {
			_, _ = db.Exec("DELETE FROM "+table+" WHERE hand_id=?", id)
		}
	})
	seed, genesis, persisted := coordinatorCrashFixture(t, id)
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
	encoded, err := json.Marshal(persisted)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO doudizhu_hands(
hand_id,room_id,phase,reveal_key_id,reveal_public_key_sha256,aggregate_version,
snapshot_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`,
		id, persisted.RoomID, persisted.Phase, persisted.RevealKeyID,
		persisted.RevealPublicKeySHA256[:], persisted.Version, encoded)
	if err != nil {
		t.Fatal(err)
	}
	first := newCoordinatorCrashRuntime(t, db)
	if err := first.RecoverActive(ctx, func(_ context.Context, target domain.HandID) (domain.HandSnapshot, error) {
		if target != id {
			return domain.HandSnapshot{}, application.ErrNotFound
		}
		return persisted, nil
	}); err != nil {
		t.Fatal(err)
	}
	public, err := first.PublicView(ctx, id, persisted.Seats[0].AccountID)
	if err != nil {
		t.Fatal(err)
	}
	var state livehand.PublicView
	if err := json.Unmarshal(public.Payload, &state); err != nil {
		t.Fatal(err)
	}
	if state.Phase != livehand.PhaseBidding {
		t.Fatalf("unexpected phase %s", state.Phase)
	}
	bidder := persisted.Seats[state.Bidding.FirstBidder-1].AccountID
	const requestID = "r3-clientseq-exact"
	original, err := first.BidCommittedWithSequence(ctx, id, bidder, requestID, public.Version, bidding.ScoreThree, 91)
	if err != nil || original.Version != 2 {
		t.Fatalf("initial bid: %+v %v", original, err)
	}
	persisted.Phase = domain.HandPlaying
	persisted.Version++
	encoded, err = json.Marshal(persisted)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE doudizhu_hands
SET phase=?,aggregate_version=?,snapshot_json=?,updated_at=UTC_TIMESTAMP(6)
WHERE hand_id=?`, persisted.Phase, persisted.Version, encoded, id); err != nil {
		t.Fatal(err)
	}
	store, err := mysqlstore.New(sqlDatabase{db})
	if err != nil {
		t.Fatal(err)
	}
	successor := newCoordinatorCrashRuntime(t, db)
	if err := successor.RecoverActive(ctx, store.LoadHand); err != nil {
		t.Fatal(err)
	}
	keyring, err := protection.NewStaticKeyring("sequence-root", map[string][]byte{
		"sequence-root": bytes.Repeat([]byte{4}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	protector, err := protection.New(keyring)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := application.NewService(store, &integrationClock{now: time.Now().UTC()},
		&integrationIDs{prefix: "sequence-replay"}, integrationSetup{}, integrationBeaconVerifier{},
		successor, &integrationOpener{}, protector, textnormalization.NFKC{}, application.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	// Attempt the modified sequence FIRST on the new coordinator.
	if _, err := svc.SubmitLiveHandBidWithSequence(ctx, bidder, id, requestID, public.Version, bidding.ScoreThree, 92); !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("changed sequence was accepted after restart: %v", err)
	}
	got, err := svc.SubmitLiveHandBidWithSequence(ctx, bidder, id, requestID, public.Version, bidding.ScoreThree, 91)
	if err != nil || !got.Historical || got.Version != original.Version ||
		!bytes.Equal(got.Payload, original.Payload) {
		t.Fatalf("original request was displaced: %+v, expected %+v, err %v", got, original, err)
	}
	if _, err := svc.SubmitLiveHandBidWithSequence(ctx, bidder, id, requestID, public.Version, bidding.ScoreTwo, 91); !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("changed score accepted: %v", err)
	}
	outsider := "outsider"
	if _, err := svc.SubmitLiveHandBidWithSequence(ctx, domain.AccountID(outsider), id, requestID, public.Version, bidding.ScoreThree, 91); !errors.Is(err, domain.ErrNotSeated) {
		t.Fatalf("outsider read stored receipt: %v", err)
	}
	if _, err := svc.SubmitLiveHandPassWithSequence(ctx, bidder, id, requestID, public.Version, 91); !errors.Is(err, gamecore.ErrVerificationFailed) {
		t.Fatalf("changed type accepted: %v", err)
	}
	actions, err := journal.Read(ctx, id)
	if err != nil || len(actions) != 1 || actions[0].ClientSeq != 91 || actions[0].Version != 2 {
		t.Fatalf("journal lost bound sequence or mutated state: %+v %v", actions, err)
	}
	// R4-N04: changing only the stored sequence must invalidate the immutable
	// authenticated identity, including a cold coordinator restart.
	if _, err := db.ExecContext(ctx, "UPDATE doudizhu_live_actions SET client_sequence=92 WHERE hand_id=? AND state_version=2", id); err != nil {
		t.Fatal(err)
	}
	tampered := newCoordinatorCrashRuntime(t, db)
	if err := tampered.RecoverActive(ctx, store.LoadHand); !errors.Is(err, mysqlstore.ErrLiveReplayIntegrity) {
		t.Fatalf("tampered client sequence was accepted: %v", err)
	}
}
