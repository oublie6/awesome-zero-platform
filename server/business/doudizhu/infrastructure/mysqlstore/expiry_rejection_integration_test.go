//go:build integration

package mysqlstore_test

import (
	"bytes"
	"context"
	"encoding/json"
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
)

// Round 10 regression: a business-rejected ExpireHand must persist only its
// rejection receipt, never a writer fence. The subsequent legitimate writer
// must still be able to commit, and a valid expiry must still fence atomically.
func TestRejectedExpiryDoesNotFenceLiveWriterInRealMySQL(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("real MySQL integration disabled")
	}
	db := openIntegrationDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	prefix := fmt.Sprintf("round10-expiry-%d", time.Now().UnixNano())
	roomID := domain.RoomID(prefix + "-room")
	id := domain.HandID(prefix + "-hand")
	t.Cleanup(func() {
		for _, query := range []string{
			"DELETE FROM doudizhu_live_checkpoints WHERE hand_id=?",
			"DELETE FROM doudizhu_live_actions WHERE hand_id=?",
			"DELETE FROM doudizhu_live_writers WHERE hand_id=?",
			"DELETE FROM doudizhu_protected_server_seeds WHERE hand_id=?",
		} {
			if _, err := db.Exec(query, id); err != nil {
				t.Errorf("cleanup durable hand: %v", err)
			}
		}
		cleanupIntegrationRows(t, db, prefix)
	})

	seed, genesis, biddingHand := coordinatorCrashFixture(t, id)
	genesis.RoomID, biddingHand.RoomID = roomID, roomID
	room, _, err := domain.NewRoom(roomID, "crash-player-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range []domain.AccountID{"crash-player-2", "crash-player-3"} {
		if _, err := room.Join(actor, room.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	for _, actor := range []domain.AccountID{"crash-player-1", "crash-player-2", "crash-player-3"} {
		if _, err := room.SetReady(actor, true, room.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := room.StartHand("crash-player-1", id, room.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	store, err := mysqlstore.New(sqlDatabase{db})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WithinCommand(ctx, "crash-player-1", prefix+"-fixture", func(txCtx context.Context, tx application.Transaction) error {
		if err := tx.InsertRoom(txCtx, room.Snapshot(), time.Now().UTC()); err != nil {
			return err
		}
		return tx.InsertHand(txCtx, biddingHand, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}

	protected, err := mysqlstore.NewProtectedSeedStore(db, coordinatorCrashKey())
	if err != nil {
		t.Fatal(err)
	}
	if err := protected.Save(ctx, id, seed); err != nil {
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
	live := newCoordinatorCrashRuntime(t, db)
	if err := live.RecoverActive(ctx, func(_ context.Context, target domain.HandID) (domain.HandSnapshot, error) {
		if target != id {
			return domain.HandSnapshot{}, application.ErrNotFound
		}
		return biddingHand, nil
	}); err != nil {
		t.Fatal(err)
	}
	public, err := live.PublicView(ctx, id, "crash-player-1")
	if err != nil {
		t.Fatal(err)
	}
	var state livehand.PublicView
	if err := json.Unmarshal(public.Payload, &state); err != nil {
		t.Fatal(err)
	}
	bidder := domain.AccountID(fmt.Sprintf("crash-player-%d", state.Bidding.FirstBidder))
	clock := &integrationClock{now: time.Now().UTC().Truncate(time.Millisecond)}
	keyring, err := protection.NewStaticKeyring("round10-storage", map[string][]byte{"round10-storage": bytes.Repeat([]byte{7}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	protector, err := protection.New(keyring)
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewService(store, clock, &integrationIDs{prefix: prefix},
		integrationSetup{}, integrationBeaconVerifier{}, live, &integrationOpener{}, protector,
		textnormalization.NFKC{}, application.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	var epoch, watermark uint64
	if err := db.QueryRowContext(ctx, "SELECT owner_epoch,committed_version FROM doudizhu_live_writers WHERE hand_id=?", id).
		Scan(&epoch, &watermark); err != nil {
		t.Fatal(err)
	}
	if epoch < 2 || watermark != 1 {
		t.Fatalf("invalid live writer fixture: epoch=%d watermark=%d", epoch, watermark)
	}

	stale := integrationCommand(clock.Now(), application.CommandHandExpire, domain.AggregateHand,
		string(id), prefix+"-rejected-expiry", 1, biddingHand.Version-1)
	input := application.TerminateHandInput{ReasonCode: "bidding_timeout", ExpectedLiveVersion: watermark}
	rejected, err := service.ExpireHand(ctx, "crash-player-1", stale, input)
	if err != nil || rejected.Accepted || rejected.Failure == nil || rejected.Failure.Code != application.CodeVersionConflict {
		t.Fatalf("stale expiry should be a durable business rejection: %+v %v", rejected, err)
	}
	var gotEpoch, gotWatermark uint64
	if err := db.QueryRowContext(ctx, "SELECT owner_epoch,committed_version FROM doudizhu_live_writers WHERE hand_id=?", id).
		Scan(&gotEpoch, &gotWatermark); err != nil {
		t.Fatal(err)
	}
	if gotEpoch != epoch || gotWatermark != watermark {
		t.Fatalf("business-rejected expiry fenced a live writer: epoch %d->%d watermark %d->%d", epoch, gotEpoch, watermark, gotWatermark)
	}
	handAfter, err := store.LoadHand(ctx, id)
	if err != nil || handAfter.Phase != domain.HandBidding || handAfter.Version != biddingHand.Version {
		t.Fatalf("rejected expiry mutated authoritative hand: %+v %v", handAfter, err)
	}
	duplicate, err := service.ExpireHand(ctx, "crash-player-1", stale, input)
	if err != nil || !duplicate.Duplicate || duplicate.Accepted || duplicate.Failure == nil ||
		duplicate.Failure.Code != application.CodeVersionConflict {
		t.Fatalf("rejected receipt lost exact idempotency: %+v %v", duplicate, err)
	}
	var storedAccepted bool
	if err := db.QueryRowContext(ctx, "SELECT accepted FROM doudizhu_command_results WHERE command_id=?", stale.CommandID).
		Scan(&storedAccepted); err != nil || storedAccepted {
		t.Fatalf("stale rejection not persisted: accepted=%t err=%v", storedAccepted, err)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM doudizhu_outbox_events WHERE aggregate_id=?", 0, string(id))

	advanced, err := live.BidCommitted(ctx, id, bidder, prefix+"-valid-bid", 1, bidding.ScoreOne)
	if err != nil || advanced.Version != watermark+1 {
		t.Fatalf("valid writer stopped after rejected expiry: %+v %v", advanced, err)
	}
	accepted := integrationCommand(clock.Now(), application.CommandHandExpire, domain.AggregateHand,
		string(id), prefix+"-accepted-expiry", 2, biddingHand.Version)
	result, err := service.ExpireHand(ctx, "crash-player-1", accepted,
		application.TerminateHandInput{ReasonCode: "bidding_timeout", ExpectedLiveVersion: advanced.Version})
	if err != nil || !result.Accepted {
		t.Fatalf("valid expiry failed: %+v %v", result, err)
	}
	if err := db.QueryRowContext(ctx, "SELECT owner_epoch,committed_version FROM doudizhu_live_writers WHERE hand_id=?", id).
		Scan(&gotEpoch, &gotWatermark); err != nil {
		t.Fatal(err)
	}
	if gotEpoch != epoch+1 || gotWatermark != advanced.Version {
		t.Fatalf("accepted expiry did not fence exactly once: epoch=%d->%d watermark=%d", epoch, gotEpoch, gotWatermark)
	}
	finalHand, err := store.LoadHand(ctx, id)
	if err != nil || finalHand.Phase != domain.HandExpired {
		t.Fatalf("accepted expiry not committed: %+v %v", finalHand, err)
	}
	again, err := service.ExpireHand(ctx, "crash-player-1", accepted,
		application.TerminateHandInput{ReasonCode: "bidding_timeout", ExpectedLiveVersion: advanced.Version})
	if err != nil || !again.Accepted || !again.Duplicate {
		t.Fatalf("duplicate accepted expiry not idempotent: %+v %v", again, err)
	}
	if err := db.QueryRowContext(ctx, "SELECT owner_epoch FROM doudizhu_live_writers WHERE hand_id=?", id).
		Scan(&gotEpoch); err != nil || gotEpoch != epoch+1 {
		t.Fatalf("duplicate accepted expiry re-fenced writer: %d %v", gotEpoch, err)
	}
}

func TestHistoricalArchiveLocatorDoesNotLeakToReplacementMemberWithRealMySQL(t *testing.T) {
	if os.Getenv("APP_API_INTEGRATION") != "1" {
		t.Skip("real MySQL integration disabled")
	}
	db := openIntegrationDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	prefix := fmt.Sprintf("round10-history-%d", time.Now().UnixNano())
	roomID := domain.RoomID(prefix + "-room")
	id := domain.HandID(prefix + "-hand")
	t.Cleanup(func() { cleanupIntegrationRows(t, db, prefix) })
	_, _, biddingHand := coordinatorCrashFixture(t, id)
	biddingHand.RoomID = roomID
	hand, err := domain.RestoreHand(biddingHand)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hand.Expire("historical archive", hand.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	room, _, err := domain.NewRoom(roomID, "crash-player-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range []domain.AccountID{"crash-player-2", "crash-player-3"} {
		if _, err := room.Join(actor, room.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	for _, actor := range []domain.AccountID{"crash-player-1", "crash-player-2", "crash-player-3"} {
		if _, err := room.SetReady(actor, true, room.Snapshot().Version); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := room.StartHand("crash-player-1", id, room.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if _, err := room.FinishHand(id, room.Snapshot().Version, true); err != nil {
		t.Fatal(err)
	}
	if _, err := room.Leave("crash-player-3", room.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	if _, err := room.Join("replacement-4", room.Snapshot().Version); err != nil {
		t.Fatal(err)
	}
	store, err := mysqlstore.New(sqlDatabase{db})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WithinCommand(ctx, "crash-player-1", prefix+"-fixture", func(txCtx context.Context, tx application.Transaction) error {
		if err := tx.InsertRoom(txCtx, room.Snapshot(), time.Now().UTC()); err != nil {
			return err
		}
		return tx.InsertHand(txCtx, hand.Snapshot(), time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	service, _ := newIntegrationService(t, db, prefix)
	for _, actor := range []domain.AccountID{"crash-player-1", "crash-player-2"} {
		view, err := service.GetRoomView(ctx, actor, roomID)
		if err != nil || !view.Member || view.LastHandID != string(id) {
			t.Fatalf("historical participant %s cannot recover permitted archive: %+v %v", actor, view, err)
		}
	}
	newcomer, err := service.GetRoomView(ctx, "replacement-4", roomID)
	if err != nil || !newcomer.Member || newcomer.Occupied != 3 || newcomer.LastHandID != "" ||
		newcomer.ActiveHandID != "" || len(newcomer.Seats) != 3 {
		t.Fatalf("replacement member received inaccessible prior archive: %+v %v", newcomer, err)
	}
	departed, err := service.GetRoomView(ctx, "crash-player-3", roomID)
	if err != nil || departed.Member || departed.LastHandID != "" || len(departed.Seats) != 0 {
		t.Fatalf("former room member received private view: %+v %v", departed, err)
	}
}
